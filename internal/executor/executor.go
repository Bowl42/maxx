package executor

import (
	"context"
	"log"
	"net/http"
	"time"

	ctxutil "github.com/Bowl42/maxx-next/internal/context"
	"github.com/Bowl42/maxx-next/internal/domain"
	"github.com/Bowl42/maxx-next/internal/event"
	"github.com/Bowl42/maxx-next/internal/repository"
	"github.com/Bowl42/maxx-next/internal/router"
)

// Executor handles request execution with retry logic
type Executor struct {
	router           *router.Router
	proxyRequestRepo repository.ProxyRequestRepository
	attemptRepo      repository.ProxyUpstreamAttemptRepository
	retryConfigRepo  repository.RetryConfigRepository
	broadcaster      event.Broadcaster
	instanceID       string
}

// NewExecutor creates a new executor
func NewExecutor(
	r *router.Router,
	prr repository.ProxyRequestRepository,
	ar repository.ProxyUpstreamAttemptRepository,
	rcr repository.RetryConfigRepository,
	bc event.Broadcaster,
	instanceID string,
) *Executor {
	return &Executor{
		router:           r,
		proxyRequestRepo: prr,
		attemptRepo:      ar,
		retryConfigRepo:  rcr,
		broadcaster:      bc,
		instanceID:       instanceID,
	}
}

// Execute handles the proxy request with routing and retry logic
func (e *Executor) Execute(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	clientType := ctxutil.GetClientType(ctx)
	projectID := ctxutil.GetProjectID(ctx)
	requestModel := ctxutil.GetRequestModel(ctx)

	log.Printf("[Executor] clientType=%s, projectID=%d, model=%s", clientType, projectID, requestModel)

	// Match routes
	routes, err := e.router.Match(clientType, projectID)
	if err != nil {
		log.Printf("[Executor] Route match error: %v", err)
		return domain.NewProxyErrorWithMessage(domain.ErrNoRoutes, false, "no routes available")
	}

	log.Printf("[Executor] Matched %d routes", len(routes))

	if len(routes) == 0 {
		log.Printf("[Executor] No routes configured for clientType=%s, projectID=%d", clientType, projectID)
		return domain.NewProxyErrorWithMessage(domain.ErrNoRoutes, false, "no routes configured")
	}

	// Create proxy request record
	proxyReq := &domain.ProxyRequest{
		InstanceID:   e.instanceID,
		RequestID:    generateRequestID(),
		SessionID:    ctxutil.GetSessionID(ctx),
		ClientType:   clientType,
		RequestModel: requestModel,
		StartTime:    time.Now(),
		Status:       "IN_PROGRESS",
	}
	if err := e.proxyRequestRepo.Create(proxyReq); err != nil {
		// Log but continue
	}
	ctx = ctxutil.WithProxyRequest(ctx, proxyReq)

	// Broadcast new request immediately so frontend sees it
	if e.broadcaster != nil {
		e.broadcaster.BroadcastProxyRequest(proxyReq)
	}

	// Try routes in order with retry logic
	var lastErr error
	for _, matchedRoute := range routes {
		// Determine model mapping
		mappedModel := e.mapModel(requestModel, matchedRoute.Route, matchedRoute.Provider)
		ctx = ctxutil.WithMappedModel(ctx, mappedModel)

		// Get retry config
		retryConfig := e.getRetryConfig(matchedRoute.RetryConfig)

		// Execute with retries
		for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
			// Create attempt record
			attemptRecord := &domain.ProxyUpstreamAttempt{
				ProxyRequestID: proxyReq.ID,
				RouteID:        matchedRoute.Route.ID,
				ProviderID:     matchedRoute.Provider.ID,
				Status:         "IN_PROGRESS",
			}
			if err := e.attemptRepo.Create(attemptRecord); err != nil {
				// Log but continue
			}

			// Broadcast new attempt immediately
			if e.broadcaster != nil {
				e.broadcaster.BroadcastProxyUpstreamAttempt(attemptRecord)
			}

			// Put attempt into context so adapter can populate request/response info
			attemptCtx := ctxutil.WithUpstreamAttempt(ctx, attemptRecord)

			// Execute request
			err := matchedRoute.ProviderAdapter.Execute(attemptCtx, w, req, matchedRoute.Provider)
			if err == nil {
				// Success
				attemptRecord.Status = "COMPLETED"
				_ = e.attemptRepo.Update(attemptRecord)
				if e.broadcaster != nil {
					e.broadcaster.BroadcastProxyUpstreamAttempt(attemptRecord)
				}
				proxyReq.Status = "COMPLETED"
				proxyReq.EndTime = time.Now()
				proxyReq.Duration = proxyReq.EndTime.Sub(proxyReq.StartTime)
				proxyReq.FinalProxyUpstreamAttemptID = attemptRecord.ID
				_ = e.proxyRequestRepo.Update(proxyReq)

				// Broadcast to WebSocket clients
				if e.broadcaster != nil {
					e.broadcaster.BroadcastProxyRequest(proxyReq)
				}

				return nil
			}

			// Handle error
			lastErr = err
			attemptRecord.Status = "FAILED"
			_ = e.attemptRepo.Update(attemptRecord)
			if e.broadcaster != nil {
				e.broadcaster.BroadcastProxyUpstreamAttempt(attemptRecord)
			}
			proxyReq.ProxyUpstreamAttemptCount++

			// Check if retryable
			proxyErr, ok := err.(*domain.ProxyError)
			if !ok || !proxyErr.Retryable {
				break // Move to next route
			}

			// Wait before retry (unless last attempt)
			if attempt < retryConfig.MaxRetries {
				waitTime := e.calculateBackoff(retryConfig, attempt)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(waitTime):
				}
			}
		}
	}

	// All routes failed
	proxyReq.Status = "FAILED"
	proxyReq.EndTime = time.Now()
	proxyReq.Duration = proxyReq.EndTime.Sub(proxyReq.StartTime)
	if lastErr != nil {
		proxyReq.Error = lastErr.Error()
	}
	_ = e.proxyRequestRepo.Update(proxyReq)

	// Broadcast to WebSocket clients
	if e.broadcaster != nil {
		e.broadcaster.BroadcastProxyRequest(proxyReq)
	}

	if lastErr != nil {
		return lastErr
	}
	return domain.NewProxyErrorWithMessage(domain.ErrAllRoutesFailed, false, "all routes exhausted")
}

func (e *Executor) mapModel(requestModel string, route *domain.Route, provider *domain.Provider) string {
	// Route mapping takes precedence
	if route.ModelMapping != nil {
		if mapped, ok := route.ModelMapping[requestModel]; ok {
			return mapped
		}
	}

	// Provider mapping
	if provider.Config != nil {
		if provider.Config.Custom != nil && provider.Config.Custom.ModelMapping != nil {
			if mapped, ok := provider.Config.Custom.ModelMapping[requestModel]; ok {
				return mapped
			}
		}
		if provider.Config.Antigravity != nil && provider.Config.Antigravity.ModelMapping != nil {
			if mapped, ok := provider.Config.Antigravity.ModelMapping[requestModel]; ok {
				return mapped
			}
		}
	}

	// No mapping, use original
	return requestModel
}

func (e *Executor) getRetryConfig(config *domain.RetryConfig) *domain.RetryConfig {
	if config != nil {
		return config
	}

	// Get default config
	defaultConfig, err := e.retryConfigRepo.GetDefault()
	if err == nil && defaultConfig != nil {
		return defaultConfig
	}

	// Fallback to hardcoded defaults
	return &domain.RetryConfig{
		MaxRetries:      3,
		InitialInterval: time.Second,
		BackoffRate:     2.0,
		MaxInterval:     30 * time.Second,
	}
}

func (e *Executor) calculateBackoff(config *domain.RetryConfig, attempt int) time.Duration {
	wait := float64(config.InitialInterval)
	for i := 0; i < attempt; i++ {
		wait *= config.BackoffRate
	}
	if time.Duration(wait) > config.MaxInterval {
		return config.MaxInterval
	}
	return time.Duration(wait)
}

func generateRequestID() string {
	return time.Now().Format("20060102150405.000000")
}

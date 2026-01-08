package router

import (
	"log"
	"math/rand"
	"sort"
	"sync"

	"github.com/Bowl42/maxx-next/internal/adapter/provider"
	"github.com/Bowl42/maxx-next/internal/domain"
	"github.com/Bowl42/maxx-next/internal/repository/cached"
)

// MatchedRoute contains all data needed to execute a proxy request
type MatchedRoute struct {
	Route           *domain.Route
	Provider        *domain.Provider
	ProviderAdapter provider.ProviderAdapter
	RetryConfig     *domain.RetryConfig
}

// Router handles route matching and selection
type Router struct {
	routeRepo           *cached.RouteRepository
	providerRepo        *cached.ProviderRepository
	routingStrategyRepo *cached.RoutingStrategyRepository
	retryConfigRepo     *cached.RetryConfigRepository

	// Adapter cache
	adapters map[uint64]provider.ProviderAdapter
	mu       sync.RWMutex
}

// NewRouter creates a new router
func NewRouter(
	routeRepo *cached.RouteRepository,
	providerRepo *cached.ProviderRepository,
	routingStrategyRepo *cached.RoutingStrategyRepository,
	retryConfigRepo *cached.RetryConfigRepository,
) *Router {
	return &Router{
		routeRepo:           routeRepo,
		providerRepo:        providerRepo,
		routingStrategyRepo: routingStrategyRepo,
		retryConfigRepo:     retryConfigRepo,
		adapters:            make(map[uint64]provider.ProviderAdapter),
	}
}

// InitAdapters initializes adapters for all providers
func (r *Router) InitAdapters() error {
	providers := r.providerRepo.GetAll()
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, p := range providers {
		factory, ok := provider.GetAdapterFactory(p.Type)
		if !ok {
			continue // Skip providers without registered adapters
		}
		a, err := factory(p)
		if err != nil {
			return err
		}
		r.adapters[p.ID] = a
	}
	return nil
}

// RefreshAdapter refreshes the adapter for a specific provider
func (r *Router) RefreshAdapter(p *domain.Provider) error {
	factory, ok := provider.GetAdapterFactory(p.Type)
	if !ok {
		return nil
	}
	a, err := factory(p)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.adapters[p.ID] = a
	r.mu.Unlock()
	return nil
}

// RemoveAdapter removes the adapter for a provider
func (r *Router) RemoveAdapter(providerID uint64) {
	r.mu.Lock()
	delete(r.adapters, providerID)
	r.mu.Unlock()
}

// Match returns matched routes for a client type and project
func (r *Router) Match(clientType domain.ClientType, projectID uint64) ([]*MatchedRoute, error) {
	routes := r.routeRepo.GetAll()

	log.Printf("[Router] Total routes in cache: %d", len(routes))

	// Filter routes
	var filtered []*domain.Route
	var hasProjectRoutes bool

	for _, route := range routes {
		log.Printf("[Router] Route id=%d, clientType=%s, enabled=%v, projectID=%d",
			route.ID, route.ClientType, route.IsEnabled, route.ProjectID)

		if !route.IsEnabled {
			continue
		}
		if route.ClientType != clientType {
			continue
		}
		if route.ProjectID == projectID && projectID != 0 {
			filtered = append(filtered, route)
			hasProjectRoutes = true
		}
	}

	// If no project-specific routes, use global routes
	if !hasProjectRoutes {
		for _, route := range routes {
			if !route.IsEnabled {
				continue
			}
			if route.ClientType != clientType {
				continue
			}
			if route.ProjectID == 0 {
				filtered = append(filtered, route)
			}
		}
	}

	log.Printf("[Router] Filtered routes: %d", len(filtered))

	if len(filtered) == 0 {
		return nil, domain.ErrNoRoutes
	}

	// Get routing strategy
	strategy := r.getRoutingStrategy(projectID)

	// Sort routes by strategy
	r.sortRoutes(filtered, strategy)

	// Get default retry config
	defaultRetry, _ := r.retryConfigRepo.GetDefault()

	// Build matched routes
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []*MatchedRoute
	providers := r.providerRepo.GetAll()

	log.Printf("[Router] Providers in cache: %d, Adapters: %d", len(providers), len(r.adapters))

	for _, route := range filtered {
		provider, ok := providers[route.ProviderID]
		if !ok {
			log.Printf("[Router] Provider not found for route %d (providerID=%d)", route.ID, route.ProviderID)
			continue
		}

		adp, ok := r.adapters[route.ProviderID]
		if !ok {
			log.Printf("[Router] Adapter not found for provider %d", route.ProviderID)
			continue
		}

		var retryConfig *domain.RetryConfig
		if route.RetryConfigID != 0 {
			retryConfig, _ = r.retryConfigRepo.GetByID(route.RetryConfigID)
		}
		if retryConfig == nil {
			retryConfig = defaultRetry
		}

		matched = append(matched, &MatchedRoute{
			Route:           route,
			Provider:        provider,
			ProviderAdapter: adp,
			RetryConfig:     retryConfig,
		})
	}

	log.Printf("[Router] Final matched routes: %d", len(matched))

	if len(matched) == 0 {
		return nil, domain.ErrNoRoutes
	}

	return matched, nil
}

func (r *Router) getRoutingStrategy(projectID uint64) *domain.RoutingStrategy {
	// Try project-specific strategy first
	if projectID != 0 {
		if s, err := r.routingStrategyRepo.GetByProjectID(projectID); err == nil {
			return s
		}
	}
	// Fall back to global strategy
	if s, err := r.routingStrategyRepo.GetByProjectID(0); err == nil {
		return s
	}
	// Default to priority
	return &domain.RoutingStrategy{Type: domain.RoutingStrategyPriority}
}

func (r *Router) sortRoutes(routes []*domain.Route, strategy *domain.RoutingStrategy) {
	switch strategy.Type {
	case domain.RoutingStrategyWeightedRandom:
		// Shuffle with weights (simplified - just shuffle for now)
		rand.Shuffle(len(routes), func(i, j int) {
			routes[i], routes[j] = routes[j], routes[i]
		})
	default: // priority
		sort.Slice(routes, func(i, j int) bool {
			return routes[i].Position < routes[j].Position
		})
	}
}

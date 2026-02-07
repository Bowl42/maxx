package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/awsl-project/maxx/internal/adapter/provider"
	cliproxyapi "github.com/awsl-project/maxx/internal/adapter/provider/cliproxyapi_codex"
	"github.com/awsl-project/maxx/internal/domain"
	"github.com/awsl-project/maxx/internal/flow"
	"github.com/awsl-project/maxx/internal/usage"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func init() {
	provider.RegisterAdapterFactory("codex", NewAdapter)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			codexCacheMu.Lock()
			now := time.Now()
			for k, v := range codexCaches {
				if now.After(v.Expire) {
					delete(codexCaches, k)
				}
			}
			codexCacheMu.Unlock()
		}
	}()
}

// TokenCache caches access tokens
type TokenCache struct {
	AccessToken string
	ExpiresAt   time.Time
}

// ProviderUpdateFunc is a callback to persist token updates to the provider config
type ProviderUpdateFunc func(provider *domain.Provider) error

// CodexAdapter handles communication with OpenAI Codex API
type CodexAdapter struct {
	provider       *domain.Provider
	tokenCache     *TokenCache
	tokenMu        sync.RWMutex
	httpClient     *http.Client
	providerUpdate ProviderUpdateFunc
}

// SetProviderUpdateFunc sets the callback for persisting provider updates
func (a *CodexAdapter) SetProviderUpdateFunc(fn ProviderUpdateFunc) {
	a.providerUpdate = fn
}

func NewAdapter(p *domain.Provider) (provider.ProviderAdapter, error) {
	if p.Config == nil || p.Config.Codex == nil {
		return nil, fmt.Errorf("provider %s missing codex config", p.Name)
	}

	config := p.Config.Codex

	// If UseCLIProxyAPI is enabled, directly return CLIProxyAPI adapter
	if config.UseCLIProxyAPI {
		cliproxyapiProvider := &domain.Provider{
			ID:                   p.ID,
			Name:                 p.Name,
			Type:                 "cliproxyapi-codex",
			SupportedClientTypes: p.SupportedClientTypes,
			Config: &domain.ProviderConfig{
				CLIProxyAPICodex: &domain.ProviderConfigCLIProxyAPICodex{
					Email:        config.Email,
					RefreshToken: config.RefreshToken,
					AccessToken:  config.AccessToken,
					ExpiresAt:    config.ExpiresAt,
					AccountID:    config.AccountID,
					ModelMapping: config.ModelMapping,
				},
			},
		}
		return cliproxyapi.NewAdapter(cliproxyapiProvider)
	}

	adapter := &CodexAdapter{
		provider:   p,
		tokenCache: &TokenCache{},
		httpClient: newUpstreamHTTPClient(),
	}

	// Initialize token cache from persisted config if available
	if config.AccessToken != "" && config.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, config.ExpiresAt)
		if err == nil && time.Now().Before(expiresAt) {
			adapter.tokenCache = &TokenCache{
				AccessToken: config.AccessToken,
				ExpiresAt:   expiresAt,
			}
		}
	}

	return adapter, nil
}

func (a *CodexAdapter) SupportedClientTypes() []domain.ClientType {
	return []domain.ClientType{domain.ClientTypeCodex}
}

func (a *CodexAdapter) Execute(c *flow.Ctx, provider *domain.Provider) error {
	requestBody := flow.GetRequestBody(c)
	clientWantsStream := flow.GetIsStream(c)
	request := c.Request
	ctx := context.Background()
	if request != nil {
		ctx = request.Context()
	}

	// Get access token
	accessToken, err := a.getAccessToken(ctx)
	if err != nil {
		return domain.NewProxyErrorWithMessage(err, true, "failed to get access token")
	}

	// Apply Codex CLI payload adjustments (CLIProxyAPI-aligned)
	cacheID, updatedBody := applyCodexRequestTuning(c, requestBody)
	requestBody = updatedBody

	// Build upstream URL
	upstreamURL := CodexBaseURL + "/responses"
	upstreamStream := true
	if len(requestBody) > 0 {
		if updated, err := sjson.SetBytes(requestBody, "stream", upstreamStream); err == nil {
			requestBody = updated
		}
	}

	// Create upstream request
	upstreamReq, err := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(requestBody))
	if err != nil {
		return domain.NewProxyErrorWithMessage(err, true, "failed to create upstream request")
	}

	// Apply headers with passthrough support (client headers take priority)
	config := provider.Config.Codex
	a.applyCodexHeaders(upstreamReq, request, accessToken, config.AccountID, upstreamStream, cacheID)

	// Send request info via EventChannel
	if eventChan := flow.GetEventChan(c); eventChan != nil {
		eventChan.SendRequestInfo(&domain.RequestInfo{
			Method:  upstreamReq.Method,
			URL:     upstreamURL,
			Headers: flattenHeaders(upstreamReq.Header),
			Body:    string(requestBody),
		})
	}

	// Execute request
	resp, err := a.httpClient.Do(upstreamReq)
	if err != nil {
		proxyErr := domain.NewProxyErrorWithMessage(domain.ErrUpstreamError, true, "failed to connect to upstream")
		proxyErr.IsNetworkError = true
		return proxyErr
	}
	defer resp.Body.Close()

	// Handle 401 (token expired) - refresh and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		// Invalidate token cache
		a.tokenMu.Lock()
		a.tokenCache = &TokenCache{}
		a.tokenMu.Unlock()

		// Get new token
		accessToken, err = a.getAccessToken(ctx)
		if err != nil {
			return domain.NewProxyErrorWithMessage(err, true, "failed to refresh access token")
		}

		// Retry request
		upstreamReq, reqErr := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(requestBody))
		if reqErr != nil {
			return domain.NewProxyErrorWithMessage(reqErr, false, fmt.Sprintf("failed to create retry request: %v", reqErr))
		}
		a.applyCodexHeaders(upstreamReq, request, accessToken, config.AccountID, upstreamStream, cacheID)

		resp, err = a.httpClient.Do(upstreamReq)
		if err != nil {
			proxyErr := domain.NewProxyErrorWithMessage(domain.ErrUpstreamError, true, "failed to connect to upstream after token refresh")
			proxyErr.IsNetworkError = true
			return proxyErr
		}
		defer resp.Body.Close()
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)

		// Send error response info via EventChannel
		if eventChan := flow.GetEventChan(c); eventChan != nil {
			eventChan.SendResponseInfo(&domain.ResponseInfo{
				Status:  resp.StatusCode,
				Headers: flattenHeaders(resp.Header),
				Body:    string(body),
			})
		}

		proxyErr := domain.NewProxyErrorWithMessage(
			fmt.Errorf("upstream error: %s", string(body)),
			isRetryableStatusCode(resp.StatusCode),
			fmt.Sprintf("upstream returned status %d", resp.StatusCode),
		)
		proxyErr.HTTPStatusCode = resp.StatusCode
		proxyErr.IsServerError = resp.StatusCode >= 500 && resp.StatusCode < 600

		// Handle rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			proxyErr.RateLimitInfo = &domain.RateLimitInfo{
				Type:             "rate_limit",
				QuotaResetTime:   time.Now().Add(time.Minute),
				RetryHintMessage: "Rate limited by Codex API",
				ClientType:       string(domain.ClientTypeCodex),
			}
		}

		return proxyErr
	}

	// Handle response
	if clientWantsStream {
		return a.handleStreamResponse(c, resp)
	}
	return a.handleCollectedStreamResponse(c, resp)
}

func (a *CodexAdapter) getAccessToken(ctx context.Context) (string, error) {
	// Check cache
	a.tokenMu.RLock()
	if a.tokenCache.AccessToken != "" {
		if a.tokenCache.ExpiresAt.IsZero() || time.Now().Add(60*time.Second).Before(a.tokenCache.ExpiresAt) {
			token := a.tokenCache.AccessToken
			a.tokenMu.RUnlock()
			return token, nil
		}
	}
	a.tokenMu.RUnlock()

	// Use persisted access token if present (even if expiry is unknown)
	config := a.provider.Config.Codex
	if strings.TrimSpace(config.AccessToken) != "" {
		var expiresAt time.Time
		if strings.TrimSpace(config.ExpiresAt) != "" {
			if parsed, err := time.Parse(time.RFC3339, config.ExpiresAt); err == nil {
				expiresAt = parsed
			}
		}
		a.tokenMu.Lock()
		a.tokenCache = &TokenCache{
			AccessToken: config.AccessToken,
			ExpiresAt:   expiresAt,
		}
		a.tokenMu.Unlock()

		if expiresAt.IsZero() || time.Now().Add(60*time.Second).Before(expiresAt) {
			return config.AccessToken, nil
		}
	}

	// Refresh token
	tokenResp, err := RefreshAccessToken(ctx, config.RefreshToken)
	if err != nil {
		if strings.TrimSpace(config.AccessToken) != "" {
			return config.AccessToken, nil
		}
		return "", err
	}

	// Calculate expiration time (with 60s buffer)
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	// Update cache
	a.tokenMu.Lock()
	a.tokenCache = &TokenCache{
		AccessToken: tokenResp.AccessToken,
		ExpiresAt:   expiresAt,
	}
	a.tokenMu.Unlock()

	// Persist token to database if update function is set
	if a.providerUpdate != nil {
		config.AccessToken = tokenResp.AccessToken
		config.ExpiresAt = expiresAt.Format(time.RFC3339)
		if tokenResp.RefreshToken != "" {
			config.RefreshToken = tokenResp.RefreshToken
		}
		if tokenResp.IDToken != "" {
			if claims, parseErr := ParseIDToken(tokenResp.IDToken); parseErr == nil && claims != nil {
				if v := strings.TrimSpace(claims.GetAccountID()); v != "" {
					config.AccountID = v
				}
				if v := strings.TrimSpace(claims.GetUserID()); v != "" {
					config.UserID = v
				}
				if v := strings.TrimSpace(claims.Email); v != "" {
					config.Email = v
				}
				if v := strings.TrimSpace(claims.Name); v != "" {
					config.Name = v
				}
				if v := strings.TrimSpace(claims.Picture); v != "" {
					config.Picture = v
				}
				if v := strings.TrimSpace(claims.GetPlanType()); v != "" {
					config.PlanType = v
				}
				if v := strings.TrimSpace(claims.GetSubscriptionStart()); v != "" {
					config.SubscriptionStart = v
				}
				if v := strings.TrimSpace(claims.GetSubscriptionEnd()); v != "" {
					config.SubscriptionEnd = v
				}
			}
		}
		// Best-effort: token already works in memory, log if DB update fails
		if err := a.providerUpdate(a.provider); err != nil {
			log.Printf("[Codex] failed to persist refreshed token: %v", err)
		}
	}

	return tokenResp.AccessToken, nil
}

func (a *CodexAdapter) handleNonStreamResponse(c *flow.Ctx, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.NewProxyErrorWithMessage(domain.ErrUpstreamError, true, "failed to read upstream response")
	}

	// Send events via EventChannel
	if eventChan := flow.GetEventChan(c); eventChan != nil {
		eventChan.SendResponseInfo(&domain.ResponseInfo{
			Status:  resp.StatusCode,
			Headers: flattenHeaders(resp.Header),
			Body:    string(body),
		})
		// Extract token usage from response
		if metrics := usage.ExtractFromResponse(string(body)); metrics != nil {
			eventChan.SendMetrics(&domain.AdapterMetrics{
				InputTokens:  metrics.InputTokens,
				OutputTokens: metrics.OutputTokens,
			})
		}
		// Extract model from response
		if model := extractModelFromResponse(body); model != "" {
			eventChan.SendResponseModel(model)
		}
	}

	// Copy response headers
	copyResponseHeaders(c.Writer.Header(), resp.Header)
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(body)
	return nil
}

func (a *CodexAdapter) handleCollectedStreamResponse(c *flow.Ctx, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.NewProxyErrorWithMessage(domain.ErrUpstreamError, true, "failed to read upstream response")
	}

	responsePayload := body
	if isSSEPayload(body) {
		if completed := extractCodexCompletedResponse(body); len(completed) > 0 {
			responsePayload = completed
		}
	}

	if eventChan := flow.GetEventChan(c); eventChan != nil {
		eventChan.SendResponseInfo(&domain.ResponseInfo{
			Status:  resp.StatusCode,
			Headers: flattenHeaders(resp.Header),
			Body:    string(responsePayload),
		})
		if metrics := usage.ExtractFromResponse(string(responsePayload)); metrics != nil {
			eventChan.SendMetrics(&domain.AdapterMetrics{
				InputTokens:  metrics.InputTokens,
				OutputTokens: metrics.OutputTokens,
			})
		}
		if model := extractModelFromResponse(responsePayload); model != "" {
			eventChan.SendResponseModel(model)
		}
	}

	copyResponseHeaders(c.Writer.Header(), resp.Header)
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(responsePayload)
	return nil
}

func (a *CodexAdapter) handleStreamResponse(c *flow.Ctx, resp *http.Response) error {
	eventChan := flow.GetEventChan(c)
	if eventChan != nil {
		eventChan.SendResponseInfo(&domain.ResponseInfo{
			Status:  resp.StatusCode,
			Headers: flattenHeaders(resp.Header),
			Body:    "[streaming]",
		})
	}

	copyResponseHeaders(c.Writer.Header(), resp.Header)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return domain.NewProxyErrorWithMessage(domain.ErrUpstreamError, false, "streaming not supported")
	}

	// Collect SSE for token extraction
	var sseBuffer strings.Builder
	reader := bufio.NewReader(resp.Body)
	firstChunkSent := false
	responseCompleted := false

	ctx := context.Background()
	if c.Request != nil {
		ctx = c.Request.Context()
	}
	for {
		select {
		case <-ctx.Done():
			a.sendFinalStreamEvents(eventChan, &sseBuffer, resp)
			if responseCompleted {
				return nil
			}
			return domain.NewProxyErrorWithMessage(ctx.Err(), false, "client disconnected")
		default:
		}

		line, err := reader.ReadString('\n')
		if line != "" {
			sseBuffer.WriteString(line)

			// Check for response.completed in data line
			if strings.HasPrefix(line, "data:") && strings.Contains(line, "\"response.completed\"") {
				responseCompleted = true
			}

			// Write to client
			_, writeErr := c.Writer.Write([]byte(line))
			if writeErr != nil {
				a.sendFinalStreamEvents(eventChan, &sseBuffer, resp)
				if responseCompleted {
					return nil
				}
				return domain.NewProxyErrorWithMessage(writeErr, false, "client disconnected")
			}
			flusher.Flush()

			// Track TTFT
			if !firstChunkSent {
				firstChunkSent = true
				if eventChan != nil {
					eventChan.SendFirstToken(time.Now().UnixMilli())
				}
			}
		}

		if err != nil {
			a.sendFinalStreamEvents(eventChan, &sseBuffer, resp)
			if err == io.EOF || responseCompleted {
				return nil
			}
			if ctx.Err() != nil {
				return domain.NewProxyErrorWithMessage(ctx.Err(), false, "client disconnected")
			}
			return nil
		}
	}
}

func (a *CodexAdapter) sendFinalStreamEvents(eventChan domain.AdapterEventChan, sseBuffer *strings.Builder, resp *http.Response) {
	if eventChan == nil {
		return
	}
	if sseBuffer.Len() > 0 {
		// Update response body with collected SSE
		eventChan.SendResponseInfo(&domain.ResponseInfo{
			Status:  resp.StatusCode,
			Headers: flattenHeaders(resp.Header),
			Body:    sseBuffer.String(),
		})

		// Extract token usage from stream
		if metrics := usage.ExtractFromStreamContent(sseBuffer.String()); metrics != nil {
			eventChan.SendMetrics(&domain.AdapterMetrics{
				InputTokens:  metrics.InputTokens,
				OutputTokens: metrics.OutputTokens,
			})
		}

		// Extract model from stream
		if model := extractModelFromSSE(sseBuffer.String()); model != "" {
			eventChan.SendResponseModel(model)
		}
	}
}

type codexCache struct {
	ID     string
	Expire time.Time
}

var (
	codexCacheMu sync.Mutex
	codexCaches  = map[string]codexCache{}
)

func getCodexCache(key string) (codexCache, bool) {
	codexCacheMu.Lock()
	defer codexCacheMu.Unlock()
	cache, ok := codexCaches[key]
	if !ok {
		return codexCache{}, false
	}
	if time.Now().After(cache.Expire) {
		delete(codexCaches, key)
		return codexCache{}, false
	}
	return cache, true
}

func setCodexCache(key string, cache codexCache) {
	codexCacheMu.Lock()
	codexCaches[key] = cache
	codexCacheMu.Unlock()
}

func applyCodexRequestTuning(c *flow.Ctx, body []byte) (string, []byte) {
	if len(body) == 0 {
		return "", body
	}

	origBody := flow.GetOriginalRequestBody(c)
	origType := flow.GetOriginalClientType(c)

	cacheID := ""
	if origType == domain.ClientTypeClaude && len(origBody) > 0 {
		userID := gjson.GetBytes(origBody, "metadata.user_id")
		if userID.Exists() && strings.TrimSpace(userID.String()) != "" {
			model := gjson.GetBytes(body, "model").String()
			key := model + "-" + userID.String()
			if cache, ok := getCodexCache(key); ok {
				cacheID = cache.ID
			} else {
				cacheID = uuid.NewString()
				setCodexCache(key, codexCache{
					ID:     cacheID,
					Expire: time.Now().Add(1 * time.Hour),
				})
			}
		}
	} else if len(origBody) > 0 {
		if promptKey := gjson.GetBytes(origBody, "prompt_cache_key"); promptKey.Exists() {
			cacheID = promptKey.String()
		}
	}

	if cacheID != "" {
		if updated, err := sjson.SetBytes(body, "prompt_cache_key", cacheID); err == nil {
			body = updated
		}
	}

	if updated, err := sjson.SetBytes(body, "stream", true); err == nil {
		body = updated
	}
	body, _ = sjson.DeleteBytes(body, "previous_response_id")
	body, _ = sjson.DeleteBytes(body, "prompt_cache_retention")
	body, _ = sjson.DeleteBytes(body, "safety_identifier")
	if !gjson.GetBytes(body, "instructions").Exists() {
		body, _ = sjson.SetBytes(body, "instructions", "")
	}

	return cacheID, body
}

func newUpstreamHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   20 * time.Second,
		KeepAlive: 60 * time.Second,
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConnsPerHost:   16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   20 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   600 * time.Second,
	}
}

func flattenHeaders(h http.Header) map[string]string {
	result := make(map[string]string)
	for k, v := range h {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}

func copyResponseHeaders(dst, src http.Header) {
	for k, vv := range src {
		// Skip hop-by-hop headers
		switch strings.ToLower(k) {
		case "connection", "keep-alive", "transfer-encoding", "upgrade":
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func isRetryableStatusCode(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusRequestTimeout,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return status >= 500
	}
}

func extractModelFromResponse(body []byte) string {
	var resp struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &resp); err == nil && resp.Model != "" {
		return resp.Model
	}
	return ""
}

func extractModelFromSSE(sseContent string) string {
	var lastModel string
	for _, line := range strings.Split(sseContent, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			continue
		}

		var chunk struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err == nil && chunk.Model != "" {
			lastModel = chunk.Model
		}
	}
	return lastModel
}

func isSSEPayload(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	return bytes.HasPrefix(trimmed, []byte("data:")) || bytes.HasPrefix(trimmed, []byte("event:"))
}

func extractCodexCompletedResponse(body []byte) []byte {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(nil, 52_428_800)
	for scanner.Scan() {
		line := scanner.Bytes()
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		data := bytes.TrimSpace(line[5:])
		if bytes.Equal(data, []byte("[DONE]")) {
			continue
		}
		root := gjson.ParseBytes(data)
		if root.Get("type").String() == "response.completed" {
			if resp := root.Get("response"); resp.Exists() {
				return []byte(resp.Raw)
			}
			return data
		}
	}
	return nil
}

// applyCodexHeaders applies headers for Codex API requests
// It follows the CLIProxyAPI pattern: passthrough client headers, use defaults only when missing
func (a *CodexAdapter) applyCodexHeaders(upstreamReq, clientReq *http.Request, accessToken, accountID string, stream bool, cacheID string) {
	hasAccessToken := strings.TrimSpace(accessToken) != ""

	// First, copy passthrough headers from client request (excluding hop-by-hop and auth)
	if clientReq != nil {
		for k, vv := range clientReq.Header {
			lk := strings.ToLower(k)
			// Skip hop-by-hop headers and authorization (we'll set our own)
			switch lk {
			case "connection", "keep-alive", "transfer-encoding", "upgrade",
				"host", "content-length":
				continue
			case "authorization":
				if hasAccessToken {
					continue
				}
			}
			for _, v := range vv {
				upstreamReq.Header.Add(k, v)
			}
		}
	}

	// Set required headers (these always override)
	upstreamReq.Header.Set("Content-Type", "application/json")
	if hasAccessToken {
		upstreamReq.Header.Set("Authorization", "Bearer "+accessToken)
	}
	if stream {
		upstreamReq.Header.Set("Accept", "text/event-stream")
	} else {
		upstreamReq.Header.Set("Accept", "application/json")
	}
	upstreamReq.Header.Set("Connection", "Keep-Alive")

	// Set Codex-specific headers only if client didn't provide them
	ensureHeader(upstreamReq.Header, clientReq, "Version", CodexVersion)
	ensureHeader(upstreamReq.Header, clientReq, "Openai-Beta", OpenAIBetaHeader)
	if cacheID != "" {
		upstreamReq.Header.Set("Conversation_id", cacheID)
		upstreamReq.Header.Set("Session_id", cacheID)
	} else {
		ensureHeader(upstreamReq.Header, clientReq, "Session_id", uuid.NewString())
	}
	ensureHeader(upstreamReq.Header, clientReq, "User-Agent", CodexUserAgent)
	if hasAccessToken {
		ensureHeader(upstreamReq.Header, clientReq, "Originator", CodexOriginator)
	}

	// Set account ID if available (required for OAuth auth, not for API key)
	if hasAccessToken && accountID != "" {
		upstreamReq.Header.Set("Chatgpt-Account-Id", accountID)
	}
}

// ensureHeader sets a header only if the client request doesn't already have it
func ensureHeader(dst http.Header, clientReq *http.Request, key, defaultValue string) {
	if clientReq != nil && clientReq.Header.Get(key) != "" {
		// Client provided this header, it's already copied, don't override
		return
	}
	dst.Set(key, defaultValue)
}

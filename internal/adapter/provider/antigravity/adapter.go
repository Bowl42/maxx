package antigravity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Bowl42/maxx-next/internal/adapter/provider"
	"github.com/Bowl42/maxx-next/internal/converter"
	ctxutil "github.com/Bowl42/maxx-next/internal/context"
	"github.com/Bowl42/maxx-next/internal/domain"
	"github.com/Bowl42/maxx-next/internal/usage"
)

func init() {
	provider.RegisterAdapterFactory("antigravity", NewAdapter)
}

// TokenCache caches access tokens
type TokenCache struct {
	AccessToken string
	ExpiresAt   time.Time
}

type AntigravityAdapter struct {
	provider   *domain.Provider
	converter  *converter.Registry
	tokenCache *TokenCache
	tokenMu    sync.RWMutex
}

func NewAdapter(p *domain.Provider) (provider.ProviderAdapter, error) {
	if p.Config == nil || p.Config.Antigravity == nil {
		return nil, fmt.Errorf("provider %s missing antigravity config", p.Name)
	}
	return &AntigravityAdapter{
		provider:   p,
		converter:  converter.NewRegistry(),
		tokenCache: &TokenCache{},
	}, nil
}

func (a *AntigravityAdapter) SupportedClientTypes() []domain.ClientType {
	// Antigravity natively supports Claude, OpenAI, and Gemini by converting to Gemini/v1internal API
	return []domain.ClientType{domain.ClientTypeClaude, domain.ClientTypeOpenAI, domain.ClientTypeGemini}
}

func (a *AntigravityAdapter) Execute(ctx context.Context, w http.ResponseWriter, req *http.Request, provider *domain.Provider) error {
	clientType := ctxutil.GetClientType(ctx)
	mappedModel := ctxutil.GetMappedModel(ctx)
	requestBody := ctxutil.GetRequestBody(ctx)

	// Determine if streaming
	stream := isStreamRequest(requestBody)

	// Get access token
	accessToken, err := a.getAccessToken(ctx)
	if err != nil {
		return domain.NewProxyErrorWithMessage(err, true, "failed to get access token")
	}

	// Antigravity uses Gemini format
	targetType := domain.ClientTypeGemini
	needsConversion := clientType != targetType

	// Transform request if needed
	var geminiBody []byte
	if needsConversion {
		geminiBody, err = a.converter.TransformRequest(clientType, targetType, requestBody, mappedModel, stream)
		if err != nil {
			return domain.NewProxyErrorWithMessage(domain.ErrFormatConversion, true, "failed to transform request")
		}
	} else {
		// For Gemini, unwrap CLI envelope if present
		geminiBody = unwrapGeminiCLIEnvelope(requestBody)
	}

	// Wrap request in v1internal format
	config := provider.Config.Antigravity
	upstreamBody, err := wrapV1InternalRequest(geminiBody, config.ProjectID, mappedModel)
	if err != nil {
		return domain.NewProxyErrorWithMessage(domain.ErrFormatConversion, true, "failed to wrap request for v1internal")
	}

	// Build upstream URL (v1internal endpoint)
	upstreamURL := a.buildUpstreamURL(stream)

	// Create upstream request
	upstreamReq, err := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(upstreamBody))
	if err != nil {
		return domain.NewProxyErrorWithMessage(domain.ErrUpstreamError, true, "failed to create upstream request")
	}

	// Set only the required headers (like Antigravity-Manager)
	// DO NOT copy any client headers - they may contain API keys or other sensitive data
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Authorization", "Bearer "+accessToken)
	upstreamReq.Header.Set("User-Agent", AntigravityUserAgent)

	// Capture request info for attempt record
	if attempt := ctxutil.GetUpstreamAttempt(ctx); attempt != nil {
		attempt.RequestInfo = &domain.RequestInfo{
			Method:  upstreamReq.Method,
			URL:     upstreamURL,
			Headers: flattenHeaders(upstreamReq.Header),
			Body:    string(upstreamBody),
		}
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(upstreamReq)
	if err != nil {
		return domain.NewProxyErrorWithMessage(domain.ErrUpstreamError, true, "failed to connect to upstream")
	}
	defer resp.Body.Close()

	// Check for 401 (token expired) and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		// Invalidate token cache
		a.tokenMu.Lock()
		a.tokenCache = &TokenCache{}
		a.tokenMu.Unlock()

		// Get new token
		accessToken, err = a.getAccessToken(ctx)
		if err != nil {
			return domain.NewProxyErrorWithMessage(err, true, "failed to refresh access token")
		}

		// Retry request with only required headers
		upstreamReq, _ = http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(upstreamBody))
		upstreamReq.Header.Set("Content-Type", "application/json")
		upstreamReq.Header.Set("Authorization", "Bearer "+accessToken)
		upstreamReq.Header.Set("User-Agent", AntigravityUserAgent)
		resp, err = client.Do(upstreamReq)
		if err != nil {
			return domain.NewProxyErrorWithMessage(domain.ErrUpstreamError, true, "failed to connect to upstream after token refresh")
		}
		defer resp.Body.Close()
	}

	// Check for error response
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		// Capture error response info
		if attempt := ctxutil.GetUpstreamAttempt(ctx); attempt != nil {
			attempt.ResponseInfo = &domain.ResponseInfo{
				Status:  resp.StatusCode,
				Headers: flattenHeaders(resp.Header),
				Body:    string(body),
			}
		}
		return domain.NewProxyErrorWithMessage(
			fmt.Errorf("upstream error: %s", string(body)),
			isRetryableStatusCode(resp.StatusCode),
			fmt.Sprintf("upstream returned status %d", resp.StatusCode),
		)
	}

	// Handle response
	if stream {
		return a.handleStreamResponse(ctx, w, resp, clientType, targetType, needsConversion)
	}
	return a.handleNonStreamResponse(ctx, w, resp, clientType, targetType, needsConversion)
}

func (a *AntigravityAdapter) getAccessToken(ctx context.Context) (string, error) {
	// Check cache
	a.tokenMu.RLock()
	if a.tokenCache.AccessToken != "" && time.Now().Before(a.tokenCache.ExpiresAt) {
		token := a.tokenCache.AccessToken
		a.tokenMu.RUnlock()
		return token, nil
	}
	a.tokenMu.RUnlock()

	// Refresh token
	config := a.provider.Config.Antigravity
	accessToken, expiresIn, err := refreshGoogleToken(ctx, config.RefreshToken)
	if err != nil {
		return "", err
	}

	// Cache token
	a.tokenMu.Lock()
	a.tokenCache = &TokenCache{
		AccessToken: accessToken,
		ExpiresAt:   time.Now().Add(time.Duration(expiresIn-60) * time.Second), // 60s buffer
	}
	a.tokenMu.Unlock()

	return accessToken, nil
}

func refreshGoogleToken(ctx context.Context, refreshToken string) (string, int, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", OAuthClientID)
	data.Set("client_secret", OAuthClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("token refresh failed: %s", string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, err
	}

	return result.AccessToken, result.ExpiresIn, nil
}

// v1internal endpoint (same as Antigravity-Manager)
const (
	V1InternalBaseURL = "https://cloudcode-pa.googleapis.com/v1internal"
)

func (a *AntigravityAdapter) buildUpstreamURL(stream bool) string {
	if stream {
		return fmt.Sprintf("%s:streamGenerateContent?alt=sse", V1InternalBaseURL)
	}
	return fmt.Sprintf("%s:generateContent", V1InternalBaseURL)
}

func (a *AntigravityAdapter) handleNonStreamResponse(ctx context.Context, w http.ResponseWriter, resp *http.Response, clientType, targetType domain.ClientType, needsConversion bool) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.NewProxyErrorWithMessage(domain.ErrUpstreamError, true, "failed to read upstream response")
	}

	// Unwrap v1internal response wrapper (extract "response" field)
	unwrappedBody := unwrapV1InternalResponse(body)

	// Capture response info and extract token usage
	if attempt := ctxutil.GetUpstreamAttempt(ctx); attempt != nil {
		attempt.ResponseInfo = &domain.ResponseInfo{
			Status:  resp.StatusCode,
			Headers: flattenHeaders(resp.Header),
			Body:    string(body), // Keep original for debugging
		}

		// Extract token usage from unwrapped response
		if metrics := usage.ExtractFromResponse(string(unwrappedBody)); metrics != nil {
			attempt.InputTokenCount = metrics.InputTokens
			attempt.OutputTokenCount = metrics.OutputTokens
			attempt.CacheReadCount = metrics.CacheReadCount
			attempt.CacheWriteCount = metrics.CacheCreationCount
			attempt.Cache5mWriteCount = metrics.Cache5mCreationCount
			attempt.Cache1hWriteCount = metrics.Cache1hCreationCount
		}

		// Broadcast attempt update with token info
		if bc := ctxutil.GetBroadcaster(ctx); bc != nil {
			bc.BroadcastProxyUpstreamAttempt(attempt)
		}
	}

	var responseBody []byte
	if needsConversion {
		responseBody, err = a.converter.TransformResponse(targetType, clientType, unwrappedBody)
		if err != nil {
			return domain.NewProxyErrorWithMessage(domain.ErrFormatConversion, false, "failed to transform response")
		}
	} else {
		responseBody = unwrappedBody
	}

	// Copy upstream headers (except those we override)
	copyResponseHeaders(w.Header(), resp.Header)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(responseBody)
	return nil
}

func (a *AntigravityAdapter) handleStreamResponse(ctx context.Context, w http.ResponseWriter, resp *http.Response, clientType, targetType domain.ClientType, needsConversion bool) error {
	attempt := ctxutil.GetUpstreamAttempt(ctx)

	// Capture response info (for streaming, we only capture status and headers)
	if attempt != nil {
		attempt.ResponseInfo = &domain.ResponseInfo{
			Status:  resp.StatusCode,
			Headers: flattenHeaders(resp.Header),
			Body:    "[streaming]",
		}
	}

	// Copy upstream headers (except those we override)
	copyResponseHeaders(w.Header(), resp.Header)

	// Set/override streaming headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return domain.NewProxyErrorWithMessage(domain.ErrUpstreamError, false, "streaming not supported")
	}

	var state *converter.TransformState
	if needsConversion {
		state = converter.NewTransformState()
	}

	// Collect all SSE events for response body and token extraction
	var sseBuffer strings.Builder

	// Helper to extract tokens and update attempt with final response body
	extractTokens := func() {
		if attempt != nil && sseBuffer.Len() > 0 {
			// Update response body with collected SSE content
			if attempt.ResponseInfo != nil {
				attempt.ResponseInfo.Body = sseBuffer.String()
			}
			// Extract token usage
			if metrics := usage.ExtractFromStreamContent(sseBuffer.String()); metrics != nil {
				attempt.InputTokenCount = metrics.InputTokens
				attempt.OutputTokenCount = metrics.OutputTokens
				attempt.CacheReadCount = metrics.CacheReadCount
				attempt.CacheWriteCount = metrics.CacheCreationCount
				attempt.Cache5mWriteCount = metrics.Cache5mCreationCount
				attempt.Cache1hWriteCount = metrics.Cache1hCreationCount
			}
			// Broadcast attempt update with token info
			if bc := ctxutil.GetBroadcaster(ctx); bc != nil {
				bc.BroadcastProxyUpstreamAttempt(attempt)
			}
		}
	}

	// Use buffer-based approach like Antigravity-Manager
	// Read chunks and accumulate until we have complete lines
	var lineBuffer bytes.Buffer
	buf := make([]byte, 4096)

	for {
		// Check context before reading
		select {
		case <-ctx.Done():
			extractTokens()
			return domain.NewProxyErrorWithMessage(ctx.Err(), false, "client disconnected")
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			lineBuffer.Write(buf[:n])

			// Process complete lines (lines ending with \n)
			for {
				line, readErr := lineBuffer.ReadString('\n')
				if readErr != nil {
					// No complete line yet, put partial data back
					lineBuffer.WriteString(line)
					break
				}

				// Collect for token extraction
				sseBuffer.WriteString(line)

				// Process the complete line
				lineBytes := []byte(line)

				// Unwrap v1internal SSE chunk before processing
				unwrappedLine := unwrapV1InternalSSEChunk(lineBytes)

				var output []byte
				if needsConversion {
					// Transform the chunk
					transformed, transformErr := a.converter.TransformStreamChunk(targetType, clientType, unwrappedLine, state)
					if transformErr != nil {
						continue // Skip malformed chunks
					}
					output = transformed
				} else {
					output = unwrappedLine
				}

				if len(output) > 0 {
					_, writeErr := w.Write(output)
					if writeErr != nil {
						// Client disconnected
						extractTokens()
						return domain.NewProxyErrorWithMessage(writeErr, false, "client disconnected")
					}
					flusher.Flush()
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				extractTokens()
				return nil
			}
			// Upstream connection closed - check if client is still connected
			if ctx.Err() != nil {
				extractTokens()
				return domain.NewProxyErrorWithMessage(ctx.Err(), false, "client disconnected")
			}
			extractTokens()
			return nil
		}
	}
}

// Helper functions

func isStreamRequest(body []byte) bool {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return false
	}
	stream, _ := req["stream"].(bool)
	return stream
}

// unwrapGeminiCLIEnvelope extracts the inner request from Gemini CLI envelope format
// Gemini CLI sends: {"request": {...}, "model": "..."}
// Gemini API expects just the inner request content
func unwrapGeminiCLIEnvelope(body []byte) []byte {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return body
	}

	if innerRequest, ok := data["request"]; ok {
		if unwrapped, err := json.Marshal(innerRequest); err == nil {
			return unwrapped
		}
	}

	return body
}

// wrapV1InternalRequest wraps the request body in v1internal format
// Similar to Antigravity-Manager's wrap_request function
func wrapV1InternalRequest(body []byte, projectID, model string) ([]byte, error) {
	var innerRequest map[string]interface{}
	if err := json.Unmarshal(body, &innerRequest); err != nil {
		return nil, err
	}

	// Remove model field from inner request if present (will be at top level)
	delete(innerRequest, "model")

	wrapped := map[string]interface{}{
		"project":     projectID,
		"requestId":   fmt.Sprintf("agent-%d", time.Now().UnixNano()),
		"request":     innerRequest,
		"model":       model,
		"userAgent":   "antigravity",
		"requestType": "agent", // Must be "agent", "web_search", or "image_gen" (not "gemini")
	}

	return json.Marshal(wrapped)
}

// unwrapV1InternalResponse extracts the response from v1internal wrapper
func unwrapV1InternalResponse(body []byte) []byte {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return body
	}

	if response, ok := data["response"]; ok {
		if unwrapped, err := json.Marshal(response); err == nil {
			return unwrapped
		}
	}

	return body
}

// unwrapV1InternalSSEChunk unwraps a single SSE chunk from v1internal format
// Input: "data: {"response": {...}}\n"
// Output: "data: {...}\n\n" (with double newline for proper SSE format)
// Returns nil for empty lines (they are already handled by \n\n terminator)
func unwrapV1InternalSSEChunk(line []byte) []byte {
	lineStr := strings.TrimSpace(string(line))

	// Skip empty lines - we already add \n\n after each data line
	if lineStr == "" {
		return nil
	}

	// Non-data lines pass through with proper SSE terminator
	if !strings.HasPrefix(lineStr, "data: ") {
		return []byte(lineStr + "\n\n")
	}

	jsonPart := strings.TrimPrefix(lineStr, "data: ")

	// Non-JSON data passes through with proper SSE terminator
	if !strings.HasPrefix(jsonPart, "{") {
		return []byte(lineStr + "\n\n")
	}

	// Try to parse and extract response field
	var wrapper map[string]interface{}
	if err := json.Unmarshal([]byte(jsonPart), &wrapper); err != nil {
		return []byte(lineStr + "\n\n")
	}

	// Extract "response" field if present (v1internal wraps response)
	if response, ok := wrapper["response"]; ok {
		if unwrapped, err := json.Marshal(response); err == nil {
			return []byte("data: " + string(unwrapped) + "\n\n")
		}
	}

	// No response field - pass through with proper SSE terminator
	return []byte(lineStr + "\n\n")
}

// Response headers to exclude when copying
var excludedResponseHeaders = map[string]bool{
	"content-length":    true,
	"transfer-encoding": true,
	"connection":        true,
	"keep-alive":        true,
}

// copyResponseHeaders copies response headers from upstream, excluding certain headers
func copyResponseHeaders(dst, src http.Header) {
	if src == nil {
		return
	}
	for key, values := range src {
		lowerKey := strings.ToLower(key)
		if excludedResponseHeaders[lowerKey] {
			continue
		}
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}

// flattenHeaders converts http.Header to map[string]string (first value only)
func flattenHeaders(h http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range h {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}

// isRetryableStatusCode returns true if the status code indicates a retryable error
func isRetryableStatusCode(code int) bool {
	switch code {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError,    // 500
		http.StatusBadGateway,             // 502
		http.StatusServiceUnavailable,     // 503
		http.StatusGatewayTimeout:         // 504
		return true
	default:
		return false
	}
}
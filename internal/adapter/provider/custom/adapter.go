package custom

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Bowl42/maxx-next/internal/adapter/provider"
	"github.com/Bowl42/maxx-next/internal/converter"
	ctxutil "github.com/Bowl42/maxx-next/internal/context"
	"github.com/Bowl42/maxx-next/internal/domain"
)

func init() {
	provider.RegisterAdapterFactory("custom", NewAdapter)
}

type CustomAdapter struct {
	provider  *domain.Provider
	converter *converter.Registry
}

func NewAdapter(p *domain.Provider) (provider.ProviderAdapter, error) {
	if p.Config == nil || p.Config.Custom == nil {
		return nil, fmt.Errorf("provider %s missing custom config", p.Name)
	}
	return &CustomAdapter{
		provider:  p,
		converter: converter.NewRegistry(),
	}, nil
}

func (a *CustomAdapter) SupportedClientTypes() []domain.ClientType {
	return a.provider.SupportedClientTypes
}

func (a *CustomAdapter) Execute(ctx context.Context, w http.ResponseWriter, req *http.Request, provider *domain.Provider) error {
	clientType := ctxutil.GetClientType(ctx)
	mappedModel := ctxutil.GetMappedModel(ctx)
	requestBody := ctxutil.GetRequestBody(ctx)

	// Determine if streaming
	stream := isStreamRequest(requestBody)

	// Determine target client type for the provider
	// If provider supports the client's type natively, use it directly
	// Otherwise, find a supported type and convert
	targetType := clientType
	needsConversion := false
	if !a.supportsClientType(clientType) {
		// Find a supported type (prefer OpenAI as it's most common)
		for _, supported := range a.provider.SupportedClientTypes {
			targetType = supported
			break
		}
		needsConversion = true
	}

	// Transform request if needed
	var upstreamBody []byte
	var err error
	if needsConversion {
		upstreamBody, err = a.converter.TransformRequest(clientType, targetType, requestBody, mappedModel, stream)
		if err != nil {
			return domain.NewProxyErrorWithMessage(domain.ErrFormatConversion, true, "failed to transform request")
		}
	} else {
		// Just update the model in the request
		upstreamBody, err = updateModelInBody(requestBody, mappedModel, clientType)
		if err != nil {
			return domain.NewProxyErrorWithMessage(domain.ErrFormatConversion, true, "failed to update model")
		}
	}

	// Build upstream URL
	baseURL := a.getBaseURL(targetType)
	upstreamURL := buildUpstreamURL(baseURL, targetType, stream)

	// Create upstream request
	upstreamReq, err := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(upstreamBody))
	if err != nil {
		return domain.NewProxyErrorWithMessage(domain.ErrUpstreamError, true, "failed to create upstream request")
	}

	// Set headers
	upstreamReq.Header.Set("Content-Type", "application/json")
	if a.provider.Config.Custom.APIKey != "" {
		setAuthHeader(upstreamReq, targetType, a.provider.Config.Custom.APIKey)
	}

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

func (a *CustomAdapter) supportsClientType(ct domain.ClientType) bool {
	for _, supported := range a.provider.SupportedClientTypes {
		if supported == ct {
			return true
		}
	}
	return false
}

func (a *CustomAdapter) getBaseURL(clientType domain.ClientType) string {
	config := a.provider.Config.Custom
	if url, ok := config.ClientBaseURL[clientType]; ok && url != "" {
		return url
	}
	return config.BaseURL
}

func (a *CustomAdapter) handleNonStreamResponse(ctx context.Context, w http.ResponseWriter, resp *http.Response, clientType, targetType domain.ClientType, needsConversion bool) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.NewProxyErrorWithMessage(domain.ErrUpstreamError, true, "failed to read upstream response")
	}

	// Capture response info
	if attempt := ctxutil.GetUpstreamAttempt(ctx); attempt != nil {
		attempt.ResponseInfo = &domain.ResponseInfo{
			Status:  resp.StatusCode,
			Headers: flattenHeaders(resp.Header),
			Body:    string(body),
		}
	}

	var responseBody []byte
	if needsConversion {
		responseBody, err = a.converter.TransformResponse(targetType, clientType, body)
		if err != nil {
			return domain.NewProxyErrorWithMessage(domain.ErrFormatConversion, false, "failed to transform response")
		}
	} else {
		responseBody = body
	}

	// Copy headers
	for k, v := range resp.Header {
		if k != "Content-Length" && k != "Transfer-Encoding" {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(responseBody)
	return nil
}

func (a *CustomAdapter) handleStreamResponse(ctx context.Context, w http.ResponseWriter, resp *http.Response, clientType, targetType domain.ClientType, needsConversion bool) error {
	// Capture response info (for streaming, we only capture status and headers)
	if attempt := ctxutil.GetUpstreamAttempt(ctx); attempt != nil {
		attempt.ResponseInfo = &domain.ResponseInfo{
			Status:  resp.StatusCode,
			Headers: flattenHeaders(resp.Header),
			Body:    "[streaming]",
		}
	}

	// Set streaming headers
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

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil // Connection closed
		}

		if needsConversion {
			// Transform the chunk
			transformed, err := a.converter.TransformStreamChunk(targetType, clientType, line, state)
			if err != nil {
				continue // Skip malformed chunks
			}
			if len(transformed) > 0 {
				_, _ = w.Write(transformed)
				flusher.Flush()
			}
		} else {
			_, _ = w.Write(line)
			flusher.Flush()
		}
	}

	return nil
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

func updateModelInBody(body []byte, model string, clientType domain.ClientType) ([]byte, error) {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	req["model"] = model
	return json.Marshal(req)
}

func buildUpstreamURL(baseURL string, clientType domain.ClientType, stream bool) string {
	baseURL = strings.TrimSuffix(baseURL, "/")

	switch clientType {
	case domain.ClientTypeClaude:
		return baseURL + "/v1/messages"
	case domain.ClientTypeOpenAI:
		return baseURL + "/v1/chat/completions"
	case domain.ClientTypeCodex:
		return baseURL + "/v1/responses"
	case domain.ClientTypeGemini:
		// Gemini uses different endpoints for stream vs non-stream
		if stream {
			return baseURL + ":streamGenerateContent?alt=sse"
		}
		return baseURL + ":generateContent"
	default:
		return baseURL + "/v1/chat/completions"
	}
}

func setAuthHeader(req *http.Request, clientType domain.ClientType, apiKey string) {
	switch clientType {
	case domain.ClientTypeClaude:
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case domain.ClientTypeGemini:
		// Gemini uses query param, but we can also set header
		req.Header.Set("x-goog-api-key", apiKey)
	default:
		// OpenAI and Codex use Bearer token
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}

func isRetryableStatusCode(code int) bool {
	switch code {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
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

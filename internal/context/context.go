package context

import (
	"context"

	"github.com/Bowl42/maxx-next/internal/domain"
)

type contextKey string

const (
	CtxKeyClientType     contextKey = "client_type"
	CtxKeySessionID      contextKey = "session_id"
	CtxKeyProjectID      contextKey = "project_id"
	CtxKeyRequestModel   contextKey = "request_model"
	CtxKeyMappedModel    contextKey = "mapped_model"
	CtxKeyResponseModel  contextKey = "response_model"
	CtxKeyProxyRequest   contextKey = "proxy_request"
	CtxKeyRequestBody    contextKey = "request_body"
	CtxKeyUpstreamAttempt contextKey = "upstream_attempt"
)

// Setters
func WithClientType(ctx context.Context, ct domain.ClientType) context.Context {
	return context.WithValue(ctx, CtxKeyClientType, ct)
}

func WithSessionID(ctx context.Context, sid string) context.Context {
	return context.WithValue(ctx, CtxKeySessionID, sid)
}

func WithProjectID(ctx context.Context, pid uint64) context.Context {
	return context.WithValue(ctx, CtxKeyProjectID, pid)
}

func WithRequestModel(ctx context.Context, model string) context.Context {
	return context.WithValue(ctx, CtxKeyRequestModel, model)
}

func WithMappedModel(ctx context.Context, model string) context.Context {
	return context.WithValue(ctx, CtxKeyMappedModel, model)
}

func WithResponseModel(ctx context.Context, model string) context.Context {
	return context.WithValue(ctx, CtxKeyResponseModel, model)
}

func WithProxyRequest(ctx context.Context, pr *domain.ProxyRequest) context.Context {
	return context.WithValue(ctx, CtxKeyProxyRequest, pr)
}

func WithRequestBody(ctx context.Context, body []byte) context.Context {
	return context.WithValue(ctx, CtxKeyRequestBody, body)
}

func WithUpstreamAttempt(ctx context.Context, attempt *domain.ProxyUpstreamAttempt) context.Context {
	return context.WithValue(ctx, CtxKeyUpstreamAttempt, attempt)
}

// Getters
func GetClientType(ctx context.Context) domain.ClientType {
	if v, ok := ctx.Value(CtxKeyClientType).(domain.ClientType); ok {
		return v
	}
	return ""
}

func GetSessionID(ctx context.Context) string {
	if v, ok := ctx.Value(CtxKeySessionID).(string); ok {
		return v
	}
	return ""
}

func GetProjectID(ctx context.Context) uint64 {
	if v, ok := ctx.Value(CtxKeyProjectID).(uint64); ok {
		return v
	}
	return 0
}

func GetRequestModel(ctx context.Context) string {
	if v, ok := ctx.Value(CtxKeyRequestModel).(string); ok {
		return v
	}
	return ""
}

func GetMappedModel(ctx context.Context) string {
	if v, ok := ctx.Value(CtxKeyMappedModel).(string); ok {
		return v
	}
	return ""
}

func GetResponseModel(ctx context.Context) string {
	if v, ok := ctx.Value(CtxKeyResponseModel).(string); ok {
		return v
	}
	return ""
}

func GetProxyRequest(ctx context.Context) *domain.ProxyRequest {
	if v, ok := ctx.Value(CtxKeyProxyRequest).(*domain.ProxyRequest); ok {
		return v
	}
	return nil
}

func GetRequestBody(ctx context.Context) []byte {
	if v, ok := ctx.Value(CtxKeyRequestBody).([]byte); ok {
		return v
	}
	return nil
}

func GetUpstreamAttempt(ctx context.Context) *domain.ProxyUpstreamAttempt {
	if v, ok := ctx.Value(CtxKeyUpstreamAttempt).(*domain.ProxyUpstreamAttempt); ok {
		return v
	}
	return nil
}

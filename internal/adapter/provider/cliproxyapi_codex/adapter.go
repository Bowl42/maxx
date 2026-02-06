package cliproxyapi_codex

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/awsl-project/maxx/internal/adapter/provider"
	ctxutil "github.com/awsl-project/maxx/internal/context"
	"github.com/awsl-project/maxx/internal/domain"
	"github.com/awsl-project/maxx/internal/usage"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/exec"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/translator"
)

type CLIProxyAPICodexAdapter struct {
	provider *domain.Provider
	authObj  *auth.Auth
	executor *exec.CodexExecutor
}

func NewAdapter(p *domain.Provider) (provider.ProviderAdapter, error) {
	if p.Config == nil || p.Config.CLIProxyAPICodex == nil {
		return nil, fmt.Errorf("provider %s missing cliproxyapi-codex config", p.Name)
	}

	// 创建 Auth 对象，executor 内部会自动处理 token 刷新
	authObj := &auth.Auth{
		Provider: "codex",
		Metadata: map[string]any{
			"type":          "codex",
			"refresh_token": p.Config.CLIProxyAPICodex.RefreshToken,
		},
	}

	adapter := &CLIProxyAPICodexAdapter{
		provider: p,
		authObj:  authObj,
		executor: exec.NewCodexExecutor(),
	}

	return adapter, nil
}

func (a *CLIProxyAPICodexAdapter) SupportedClientTypes() []domain.ClientType {
	return []domain.ClientType{domain.ClientTypeCodex}
}

func (a *CLIProxyAPICodexAdapter) Execute(ctx context.Context, w http.ResponseWriter, req *http.Request, p *domain.Provider) error {
	requestBody := ctxutil.GetRequestBody(ctx)
	stream := ctxutil.GetIsStream(ctx)
	model := ctxutil.GetMappedModel(ctx)

	// Codex CLI 使用 OpenAI Responses API 格式
	sourceFormat := translator.FormatCodex

	// 发送事件
	if eventChan := ctxutil.GetEventChan(ctx); eventChan != nil {
		eventChan.SendRequestInfo(&domain.RequestInfo{
			Method: "POST",
			URL:    fmt.Sprintf("cliproxyapi://codex/%s", model),
			Body:   string(requestBody),
		})
	}

	// 构建 executor 请求
	execReq := executor.Request{
		Model:   model,
		Payload: requestBody,
		Format:  sourceFormat,
	}

	execOpts := executor.Options{
		Stream:          stream,
		OriginalRequest: requestBody,
		SourceFormat:    sourceFormat,
	}

	if stream {
		return a.executeStream(ctx, w, execReq, execOpts)
	}
	return a.executeNonStream(ctx, w, execReq, execOpts)
}

func (a *CLIProxyAPICodexAdapter) executeNonStream(ctx context.Context, w http.ResponseWriter, execReq executor.Request, execOpts executor.Options) error {
	resp, err := a.executor.Execute(ctx, a.authObj, execReq, execOpts)
	if err != nil {
		return domain.NewProxyErrorWithMessage(err, true, fmt.Sprintf("executor request failed: %v", err))
	}

	// Extract and send token usage metrics
	if eventChan := ctxutil.GetEventChan(ctx); eventChan != nil {
		if metrics := usage.ExtractFromResponse(string(resp.Payload)); metrics != nil {
			// Adjust for Codex: input_tokens includes cached_tokens
			metrics = usage.AdjustForClientType(metrics, domain.ClientTypeCodex)
			eventChan.SendMetrics(&domain.AdapterMetrics{
				InputTokens:  metrics.InputTokens,
				OutputTokens: metrics.OutputTokens,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.Payload)

	return nil
}

func (a *CLIProxyAPICodexAdapter) executeStream(ctx context.Context, w http.ResponseWriter, execReq executor.Request, execOpts executor.Options) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return a.executeNonStream(ctx, w, execReq, execOpts)
	}

	stream, err := a.executor.ExecuteStream(ctx, a.authObj, execReq, execOpts)
	if err != nil {
		return domain.NewProxyErrorWithMessage(err, true, fmt.Sprintf("executor stream request failed: %v", err))
	}

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Collect SSE content for token extraction
	var sseBuffer bytes.Buffer

	for chunk := range stream {
		if chunk.Err != nil {
			log.Printf("[CLIProxyAPI-Codex] stream chunk error: %v", chunk.Err)
			break
		}
		if len(chunk.Payload) > 0 {
			// Collect for token extraction
			sseBuffer.Write(chunk.Payload)
			sseBuffer.WriteByte('\n')

			_, _ = w.Write(chunk.Payload)
			_, _ = w.Write([]byte("\n"))
			flusher.Flush()
		}
	}

	// Extract and send token usage metrics from collected SSE content
	if eventChan := ctxutil.GetEventChan(ctx); eventChan != nil {
		if metrics := usage.ExtractFromStreamContent(sseBuffer.String()); metrics != nil {
			// Adjust for Codex: input_tokens includes cached_tokens
			metrics = usage.AdjustForClientType(metrics, domain.ClientTypeCodex)
			eventChan.SendMetrics(&domain.AdapterMetrics{
				InputTokens:  metrics.InputTokens,
				OutputTokens: metrics.OutputTokens,
			})
		}
	}

	return nil
}

package kiro

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// GetUsageLimitsURL 获取使用限制的 API URL
	GetUsageLimitsURL = "https://codewhisperer.us-east-1.amazonaws.com/getUsageLimits"
)

// GetCachedUsage 获取缓存的 usage 数据（不会触发 API 调用）
// 如果没有缓存，返回 nil
func (a *KiroAdapter) GetCachedUsage() *UsageLimits {
	a.usageMu.RLock()
	defer a.usageMu.RUnlock()

	if a.usageCache == nil || a.usageCache.UsageLimits == nil {
		return nil
	}
	return a.usageCache.UsageLimits
}

// GetCachedUsageInfo 获取缓存的简化额度信息（不会触发 API 调用）
func (a *KiroAdapter) GetCachedUsageInfo() *UsageInfo {
	limits := a.GetCachedUsage()
	if limits == nil {
		return nil
	}
	return CalculateUsageInfo(limits)
}

// GetUsageCacheTime 获取缓存时间（用于前端显示"上次更新时间"）
func (a *KiroAdapter) GetUsageCacheTime() *time.Time {
	a.usageMu.RLock()
	defer a.usageMu.RUnlock()

	if a.usageCache == nil || a.usageCache.UsageLimits == nil {
		return nil
	}
	return &a.usageCache.CachedAt
}

// RefreshUsage 手动刷新 usage（唯一会调用 API 的方法）
// 只在用户主动点击刷新时调用，平时不会自动调用
func (a *KiroAdapter) RefreshUsage(ctx context.Context) (*UsageLimits, error) {
	limits, err := a.fetchUsageLimits(ctx)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	a.usageMu.Lock()
	a.usageCache = &UsageCache{
		UsageLimits: limits,
		CachedAt:    time.Now(),
	}
	a.usageMu.Unlock()

	return limits, nil
}

// fetchUsageLimits 实际获取 usage limits（内部方法）
func (a *KiroAdapter) fetchUsageLimits(ctx context.Context) (*UsageLimits, error) {
	// 获取 access token
	accessToken, err := a.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// 构建请求 URL
	params := url.Values{}
	params.Add("isEmailRequired", "true")
	params.Add("origin", "AI_EDITOR")
	params.Add("resourceType", "AGENTIC_REQUEST")

	requestURL := fmt.Sprintf("%s?%s", GetUsageLimitsURL, params.Encode())

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create usage limits request: %w", err)
	}

	// 设置请求头 (匹配 kiro2api/auth/usage_checker.go:44-50)
	req.Header.Set("x-amz-user-agent", "aws-sdk-js/1.0.0 KiroIDE-0.2.13-66c23a8c5d15afabec89ef9954ef52a119f10d369df04d548fc6c1eac694b0d1")
	req.Header.Set("user-agent", "aws-sdk-js/1.0.0 ua/2.1 os/darwin#24.6.0 lang/js md/nodejs#20.16.0 api/codewhispererruntime#1.0.0 m/E KiroIDE-0.2.13-66c23a8c5d15afabec89ef9954ef52a119f10d369df04d548fc6c1eac694b0d1")
	req.Header.Set("host", "codewhisperer.us-east-1.amazonaws.com")
	req.Header.Set("amz-sdk-invocation-id", generateUsageInvocationID())
	req.Header.Set("amz-sdk-request", "attempt=1; max=1")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Connection", "close")

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("usage limits request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read usage limits response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("usage limits check failed: status %d, response: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var usageLimits UsageLimits
	if err := FastUnmarshal(body, &usageLimits); err != nil {
		return nil, fmt.Errorf("failed to parse usage limits response: %w", err)
	}

	return &usageLimits, nil
}

// GetUsageInfo 获取简化的额度信息（使用缓存，不触发 API）
func (a *KiroAdapter) GetUsageInfo() *UsageInfo {
	return a.GetCachedUsageInfo()
}

// RefreshUsageInfo 手动刷新并获取简化的额度信息
func (a *KiroAdapter) RefreshUsageInfo(ctx context.Context) (*UsageInfo, error) {
	limits, err := a.RefreshUsage(ctx)
	if err != nil {
		return nil, err
	}
	return CalculateUsageInfo(limits), nil
}

// ClearUsageCache 清除 usage 缓存
func (a *KiroAdapter) ClearUsageCache() {
	a.usageMu.Lock()
	a.usageCache = &UsageCache{}
	a.usageMu.Unlock()
}

// generateUsageInvocationID 生成请求 ID
func generateUsageInvocationID() string {
	return fmt.Sprintf("%d-maxx", time.Now().UnixNano())
}

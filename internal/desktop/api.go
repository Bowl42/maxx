package desktop

import (
	"github.com/Bowl42/maxx-next/internal/domain"
	"github.com/Bowl42/maxx-next/internal/service"
)

// ===== Provider API =====

func (a *DesktopApp) GetProviders() ([]*domain.Provider, error) {
	return a.components.AdminService.GetProviders()
}

func (a *DesktopApp) GetProvider(id uint64) (*domain.Provider, error) {
	return a.components.AdminService.GetProvider(id)
}

func (a *DesktopApp) CreateProvider(provider *domain.Provider) error {
	return a.components.AdminService.CreateProvider(provider)
}

func (a *DesktopApp) UpdateProvider(provider *domain.Provider) error {
	return a.components.AdminService.UpdateProvider(provider)
}

func (a *DesktopApp) DeleteProvider(id uint64) error {
	return a.components.AdminService.DeleteProvider(id)
}

func (a *DesktopApp) ExportProviders() ([]*domain.Provider, error) {
	return a.components.AdminService.ExportProviders()
}

func (a *DesktopApp) ImportProviders(providers []*domain.Provider) (*service.ImportResult, error) {
	return a.components.AdminService.ImportProviders(providers)
}

// ===== Project API =====

func (a *DesktopApp) GetProjects() ([]*domain.Project, error) {
	return a.components.AdminService.GetProjects()
}

func (a *DesktopApp) GetProject(id uint64) (*domain.Project, error) {
	return a.components.AdminService.GetProject(id)
}

func (a *DesktopApp) GetProjectBySlug(slug string) (*domain.Project, error) {
	return a.components.AdminService.GetProjectBySlug(slug)
}

func (a *DesktopApp) CreateProject(project *domain.Project) error {
	return a.components.AdminService.CreateProject(project)
}

func (a *DesktopApp) UpdateProject(project *domain.Project) error {
	return a.components.AdminService.UpdateProject(project)
}

func (a *DesktopApp) DeleteProject(id uint64) error {
	return a.components.AdminService.DeleteProject(id)
}

// ===== Route API =====

func (a *DesktopApp) GetRoutes() ([]*domain.Route, error) {
	return a.components.AdminService.GetRoutes()
}

func (a *DesktopApp) GetRoute(id uint64) (*domain.Route, error) {
	return a.components.AdminService.GetRoute(id)
}

func (a *DesktopApp) CreateRoute(route *domain.Route) error {
	return a.components.AdminService.CreateRoute(route)
}

func (a *DesktopApp) UpdateRoute(route *domain.Route) error {
	return a.components.AdminService.UpdateRoute(route)
}

func (a *DesktopApp) DeleteRoute(id uint64) error {
	return a.components.AdminService.DeleteRoute(id)
}

// ===== Session API =====

func (a *DesktopApp) GetSessions() ([]*domain.Session, error) {
	return a.components.AdminService.GetSessions()
}

func (a *DesktopApp) UpdateSessionProject(sessionID string, projectID uint64) (*service.UpdateSessionProjectResult, error) {
	return a.components.AdminService.UpdateSessionProject(sessionID, projectID)
}

// ===== RetryConfig API =====

func (a *DesktopApp) GetRetryConfigs() ([]*domain.RetryConfig, error) {
	return a.components.AdminService.GetRetryConfigs()
}

func (a *DesktopApp) GetRetryConfig(id uint64) (*domain.RetryConfig, error) {
	return a.components.AdminService.GetRetryConfig(id)
}

func (a *DesktopApp) CreateRetryConfig(config *domain.RetryConfig) error {
	return a.components.AdminService.CreateRetryConfig(config)
}

func (a *DesktopApp) UpdateRetryConfig(config *domain.RetryConfig) error {
	return a.components.AdminService.UpdateRetryConfig(config)
}

func (a *DesktopApp) DeleteRetryConfig(id uint64) error {
	return a.components.AdminService.DeleteRetryConfig(id)
}

// ===== RoutingStrategy API =====

func (a *DesktopApp) GetRoutingStrategies() ([]*domain.RoutingStrategy, error) {
	return a.components.AdminService.GetRoutingStrategies()
}

func (a *DesktopApp) GetRoutingStrategy(id uint64) (*domain.RoutingStrategy, error) {
	return a.components.AdminService.GetRoutingStrategy(id)
}

func (a *DesktopApp) CreateRoutingStrategy(strategy *domain.RoutingStrategy) error {
	return a.components.AdminService.CreateRoutingStrategy(strategy)
}

func (a *DesktopApp) UpdateRoutingStrategy(strategy *domain.RoutingStrategy) error {
	return a.components.AdminService.UpdateRoutingStrategy(strategy)
}

func (a *DesktopApp) DeleteRoutingStrategy(id uint64) error {
	return a.components.AdminService.DeleteRoutingStrategy(id)
}

// ===== ProxyRequest API =====

func (a *DesktopApp) GetProxyRequests(limit, offset int) ([]*domain.ProxyRequest, error) {
	return a.components.AdminService.GetProxyRequests(limit, offset)
}

func (a *DesktopApp) GetProxyRequestsCursor(limit int, before, after uint64) (*service.CursorPaginationResult, error) {
	return a.components.AdminService.GetProxyRequestsCursor(limit, before, after)
}

func (a *DesktopApp) GetProxyRequestsCount() (int64, error) {
	return a.components.AdminService.GetProxyRequestsCount()
}

func (a *DesktopApp) GetProxyRequest(id uint64) (*domain.ProxyRequest, error) {
	return a.components.AdminService.GetProxyRequest(id)
}

func (a *DesktopApp) GetProxyUpstreamAttempts(proxyRequestID uint64) ([]*domain.ProxyUpstreamAttempt, error) {
	return a.components.AdminService.GetProxyUpstreamAttempts(proxyRequestID)
}

func (a *DesktopApp) GetProviderStats(clientType string, projectID uint64) (map[uint64]*domain.ProviderStats, error) {
	return a.components.AdminService.GetProviderStats(clientType, projectID)
}

// ===== Settings API =====

func (a *DesktopApp) GetSettings() (map[string]string, error) {
	return a.components.AdminService.GetSettings()
}

func (a *DesktopApp) GetSetting(key string) (string, error) {
	return a.components.AdminService.GetSetting(key)
}

func (a *DesktopApp) UpdateSetting(key, value string) error {
	return a.components.AdminService.UpdateSetting(key, value)
}

func (a *DesktopApp) DeleteSetting(key string) error {
	return a.components.AdminService.DeleteSetting(key)
}

// ===== Proxy Status API =====

func (a *DesktopApp) GetProxyStatus() *service.ProxyStatus {
	return a.components.AdminService.GetProxyStatus()
}

// ===== Logs API =====

func (a *DesktopApp) GetLogs(limit int) (*service.LogsResult, error) {
	return a.components.AdminService.GetLogs(limit)
}

// ===== Antigravity API =====

// Note: These methods are not in AdminService yet, need to add them
// For now, return placeholder implementations

type AntigravityTokenValidationResult struct {
	Valid  bool   `json:"valid"`
	Email  string `json:"email,omitempty"`
	Error  string `json:"error,omitempty"`
}

type AntigravityBatchValidationResult struct {
	Results []AntigravityTokenValidationResult `json:"results"`
}

type AntigravityQuotaData struct {
	Email           string                        `json:"email"`
	Name            string                        `json:"name"`
	Picture         string                        `json:"picture"`
	ProjectID       string                        `json:"projectId"`
	SubscriptionTier string                        `json:"subscriptionTier"`
	IsForbidden      bool                        `json:"isForbidden"`
	Models          []domain.AntigravityModelQuota `json:"models"`
	LastUpdated     int64                        `json:"lastUpdated"`
}

func (a *DesktopApp) ValidateAntigravityToken(refreshToken string) (*AntigravityTokenValidationResult, error) {
	// Placeholder - will be implemented by adding to AdminService
	return &AntigravityTokenValidationResult{
		Valid: false,
		Error: "not implemented",
	}, nil
}

func (a *DesktopApp) ValidateAntigravityTokens(tokens []string) (*AntigravityBatchValidationResult, error) {
	// Placeholder - will be implemented by adding to AdminService
	return &AntigravityBatchValidationResult{
		Results: []AntigravityTokenValidationResult{},
	}, nil
}

func (a *DesktopApp) ValidateAntigravityTokenText(tokenText string) (*AntigravityBatchValidationResult, error) {
	// Placeholder - will be implemented by adding to AdminService
	return &AntigravityBatchValidationResult{
		Results: []AntigravityTokenValidationResult{},
	}, nil
}

func (a *DesktopApp) GetAntigravityProviderQuota(providerID uint64, forceRefresh bool) (*AntigravityQuotaData, error) {
	// Placeholder - will be implemented by adding to AdminService
	return &AntigravityQuotaData{}, nil
}

// ===== Cooldown API =====

func (a *DesktopApp) GetCooldowns() ([]*domain.Cooldown, error) {
	return a.components.Router.GetCooldowns()
}

func (a *DesktopApp) ClearCooldown(providerID uint64) error {
	return a.components.Router.ClearCooldown(providerID)
}

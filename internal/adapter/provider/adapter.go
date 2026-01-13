package provider

import (
	"context"
	"net/http"

	"github.com/awsl-project/maxx/internal/domain"
)

// ProviderAdapter handles communication with upstream providers
type ProviderAdapter interface {
	// SupportedClientTypes returns the list of client types this adapter natively supports
	SupportedClientTypes() []domain.ClientType

	// Execute performs the proxy request to the upstream provider
	// It reads from ctx for ClientType, MappedModel, RequestBody
	// It writes the response to w
	// Returns ProxyError on failure
	Execute(ctx context.Context, w http.ResponseWriter, req *http.Request, provider *domain.Provider) error
}

// AdapterFactory creates ProviderAdapter instances
type AdapterFactory func(provider *domain.Provider) (ProviderAdapter, error)

// Registry holds all adapter factories
var adapterFactories = map[string]AdapterFactory{}

// RegisterAdapterFactory registers an adapter factory for a provider type
func RegisterAdapterFactory(providerType string, factory AdapterFactory) {
	adapterFactories[providerType] = factory
}

// GetAdapterFactory returns the adapter factory for a provider type
func GetAdapterFactory(providerType string) (AdapterFactory, bool) {
	f, ok := adapterFactories[providerType]
	return f, ok
}

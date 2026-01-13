package converter

import "github.com/awsl-project/maxx/internal/domain"

// Global registry instance - initialized at package level before init() functions
var globalRegistry = &Registry{
	requests:  make(map[domain.ClientType]map[domain.ClientType]RequestTransformer),
	responses: make(map[domain.ClientType]map[domain.ClientType]ResponseTransformer),
}

// RegisterConverter registers a converter pair in the global registry
func RegisterConverter(from, to domain.ClientType, req RequestTransformer, resp ResponseTransformer) {
	if globalRegistry.requests[from] == nil {
		globalRegistry.requests[from] = make(map[domain.ClientType]RequestTransformer)
	}
	if globalRegistry.responses[from] == nil {
		globalRegistry.responses[from] = make(map[domain.ClientType]ResponseTransformer)
	}
	if req != nil {
		globalRegistry.requests[from][to] = req
	}
	if resp != nil {
		globalRegistry.responses[from][to] = resp
	}
}

// registerBuiltins is called by NewRegistry to copy global registrations
func (r *Registry) registerBuiltins() {
	for from, toMap := range globalRegistry.requests {
		if r.requests[from] == nil {
			r.requests[from] = make(map[domain.ClientType]RequestTransformer)
		}
		for to, transformer := range toMap {
			r.requests[from][to] = transformer
		}
	}
	for from, toMap := range globalRegistry.responses {
		if r.responses[from] == nil {
			r.responses[from] = make(map[domain.ClientType]ResponseTransformer)
		}
		for to, transformer := range toMap {
			r.responses[from][to] = transformer
		}
	}
}

// GetGlobalRegistry returns the global registry
func GetGlobalRegistry() *Registry {
	return globalRegistry
}

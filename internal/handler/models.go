package handler

import (
	"net/http"
	"sort"
	"strings"

	"github.com/awsl-project/maxx/internal/repository"
)

// ModelsHandler serves GET /v1/models with a lightweight model list.
type ModelsHandler struct {
	responseModelRepo repository.ResponseModelRepository
	providerRepo      repository.ProviderRepository
	modelMappingRepo  repository.ModelMappingRepository
}

// NewModelsHandler creates a new ModelsHandler.
func NewModelsHandler(
	responseModelRepo repository.ResponseModelRepository,
	providerRepo repository.ProviderRepository,
	modelMappingRepo repository.ModelMappingRepository,
) *ModelsHandler {
	return &ModelsHandler{
		responseModelRepo: responseModelRepo,
		providerRepo:      providerRepo,
		modelMappingRepo:  modelMappingRepo,
	}
}

// ServeHTTP handles GET /v1/models.
func (h *ModelsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	names, err := h.collectModelNames()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	userAgent := r.Header.Get("User-Agent")
	if strings.HasPrefix(userAgent, "claude-cli") {
		writeJSON(w, http.StatusOK, buildClaudeModelsResponse(names))
		return
	}

	writeJSON(w, http.StatusOK, buildOpenAIModelsResponse(names))
}

func (h *ModelsHandler) collectModelNames() ([]string, error) {
	result := make(map[string]struct{})

	if h.responseModelRepo != nil {
		names, err := h.responseModelRepo.ListNames()
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			addModelName(result, name)
		}
	}

	if h.providerRepo != nil {
		providers, err := h.providerRepo.List()
		if err != nil {
			return nil, err
		}
		for _, provider := range providers {
			for _, name := range provider.SupportModels {
				addModelName(result, name)
			}
		}
	}

	if h.modelMappingRepo != nil {
		mappings, err := h.modelMappingRepo.ListEnabled()
		if err != nil {
			return nil, err
		}
		for _, mapping := range mappings {
			addModelName(result, mapping.TargetModel)
			addModelName(result, mapping.RequestModel)
		}
	}

	names := make([]string, 0, len(result))
	for name := range result {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func addModelName(target map[string]struct{}, name string) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return
	}
	if strings.Contains(trimmed, "*") {
		return
	}
	target[trimmed] = struct{}{}
}

func buildOpenAIModelsResponse(names []string) map[string]interface{} {
	data := make([]map[string]interface{}, 0, len(names))
	for _, name := range names {
		data = append(data, map[string]interface{}{
			"id":       name,
			"object":   "model",
			"created":  0,
			"owned_by": "maxx",
		})
	}

	return map[string]interface{}{
		"object": "list",
		"data":   data,
	}
}

func buildClaudeModelsResponse(names []string) map[string]interface{} {
	data := make([]map[string]interface{}, 0, len(names))
	for _, name := range names {
		data = append(data, map[string]interface{}{
			"id":           name,
			"display_name": name,
			"type":         "model",
		})
	}

	return map[string]interface{}{
		"data":     data,
		"has_more": false,
	}
}

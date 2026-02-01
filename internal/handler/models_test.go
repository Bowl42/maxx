package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/awsl-project/maxx/internal/domain"
)

type fakeResponseModelRepo struct {
	names []string
	err   error
}

func (f *fakeResponseModelRepo) Upsert(name string) error               { return nil }
func (f *fakeResponseModelRepo) BatchUpsert(names []string) error       { return nil }
func (f *fakeResponseModelRepo) List() ([]*domain.ResponseModel, error) { return nil, f.err }
func (f *fakeResponseModelRepo) ListNames() ([]string, error) {
	return append([]string(nil), f.names...), f.err
}

type fakeProviderRepo struct {
	providers []*domain.Provider
	err       error
}

func (f *fakeProviderRepo) Create(provider *domain.Provider) error { return nil }
func (f *fakeProviderRepo) Update(provider *domain.Provider) error { return nil }
func (f *fakeProviderRepo) Delete(id uint64) error                 { return nil }
func (f *fakeProviderRepo) GetByID(id uint64) (*domain.Provider, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeProviderRepo) List() ([]*domain.Provider, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]*domain.Provider(nil), f.providers...), nil
}

type fakeModelMappingRepo struct {
	mappings []*domain.ModelMapping
	err      error
}

func (f *fakeModelMappingRepo) Create(mapping *domain.ModelMapping) error { return nil }
func (f *fakeModelMappingRepo) Update(mapping *domain.ModelMapping) error { return nil }
func (f *fakeModelMappingRepo) Delete(id uint64) error                    { return nil }
func (f *fakeModelMappingRepo) GetByID(id uint64) (*domain.ModelMapping, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeModelMappingRepo) List() ([]*domain.ModelMapping, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]*domain.ModelMapping(nil), f.mappings...), nil
}
func (f *fakeModelMappingRepo) ListEnabled() ([]*domain.ModelMapping, error) {
	return f.List()
}
func (f *fakeModelMappingRepo) ListByClientType(clientType domain.ClientType) ([]*domain.ModelMapping, error) {
	return f.List()
}
func (f *fakeModelMappingRepo) ListByQuery(query *domain.ModelMappingQuery) ([]*domain.ModelMapping, error) {
	return f.List()
}
func (f *fakeModelMappingRepo) Count() (int, error) { return len(f.mappings), f.err }
func (f *fakeModelMappingRepo) DeleteAll() error    { return nil }
func (f *fakeModelMappingRepo) ClearAll() error     { return nil }
func (f *fakeModelMappingRepo) SeedDefaults() error { return nil }

func TestCollectModelNames(t *testing.T) {
	responseRepo := &fakeResponseModelRepo{names: []string{"gpt-1", "gpt-2"}}
	providerRepo := &fakeProviderRepo{
		providers: []*domain.Provider{
			{SupportModels: []string{"gpt-3", "*", " "}},
		},
	}
	mappingRepo := &fakeModelMappingRepo{
		mappings: []*domain.ModelMapping{
			{Pattern: "gpt-4", Target: "gpt-4o"},
			{Pattern: "gpt-*", Target: "gpt-5"},
		},
	}

	handler := NewModelsHandler(responseRepo, providerRepo, mappingRepo)
	names, err := handler.collectModelNames()
	if err != nil {
		t.Fatalf("collectModelNames error: %v", err)
	}

	want := []string{"gpt-1", "gpt-2", "gpt-3", "gpt-4", "gpt-4o", "gpt-5"}
	sort.Strings(want)
	if len(names) != len(want) {
		t.Fatalf("model count = %d, want %d", len(names), len(want))
	}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("names[%d] = %q, want %q", i, names[i], name)
		}
	}
}

func TestModelsHandlerFormats(t *testing.T) {
	responseRepo := &fakeResponseModelRepo{names: []string{"gpt-1"}}
	handler := NewModelsHandler(responseRepo, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("User-Agent", "claude-cli/2.0")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var claudeResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &claudeResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := claudeResp["has_more"]; !ok {
		t.Fatalf("claude response missing has_more")
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var openaiResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &openaiResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if openaiResp["object"] != "list" {
		t.Fatalf("openai response object = %v, want list", openaiResp["object"])
	}
}

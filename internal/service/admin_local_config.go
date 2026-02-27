package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// CodexLocalConfigSyncResult is the response payload for local Codex config sync.
type CodexLocalConfigSyncResult struct {
	Success           bool                        `json:"success"`
	BaseURL           string                      `json:"baseUrl"`
	WrittenFiles      []string                    `json:"writtenFiles"`
	Details           CodexLocalConfigSyncDetails `json:"details"`
	RecoveredAuthJSON bool                        `json:"recoveredAuthJSON,omitempty"`
	BackupFile        string                      `json:"backupFile,omitempty"`
	Message           string                      `json:"message,omitempty"`
}

// CodexLocalConfigSyncDetails reports per-file update status.
type CodexLocalConfigSyncDetails struct {
	ConfigTomlUpdated bool `json:"configTomlUpdated"`
	AuthJSONUpdated   bool `json:"authJsonUpdated"`
}

// SyncCodexLocalConfig writes merged Codex CLI config files into the current user's home folder.
func (s *AdminService) SyncCodexLocalConfig(
	r *http.Request,
	apiToken string,
	providerName string,
	model string,
) (*CodexLocalConfigSyncResult, error) {
	trimmedToken := strings.TrimSpace(apiToken)
	if trimmedToken == "" {
		return nil, fmt.Errorf("api token is required")
	}

	trimmedProvider := strings.TrimSpace(providerName)
	if trimmedProvider == "" {
		trimmedProvider = "maxx"
	}

	baseURL := deriveRequestBaseURL(r, s.serverAddr)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user home directory: %w", err)
	}

	codexDir := filepath.Join(homeDir, ".codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create codex directory: %w", err)
	}

	configTomlPath := filepath.Join(codexDir, "config.toml")
	authJSONPath := filepath.Join(codexDir, "auth.json")

	configExisting, configExists, configPerm, err := readFileIfExists(configTomlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config.toml: %w", err)
	}
	configMerged, err := mergeCodexConfigTOML(configExisting, trimmedProvider, baseURL, model)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config.toml: %w", err)
	}
	configUpdated := !configExists || !bytes.Equal(configExisting, configMerged)
	if configUpdated {
		if !configExists {
			configPerm = 0644
		}
		if err := writeFileAtomically(configTomlPath, configMerged, configPerm); err != nil {
			return nil, fmt.Errorf("failed to write config.toml: %w", err)
		}
	}

	authExisting, authExists, authPerm, err := readFileIfExists(authJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth.json: %w", err)
	}
	authMerged, recoveredAuthJSON, shouldBackupAuthJSON, err := mergeCodexAuthJSON(
		authExisting,
		trimmedProvider,
		trimmedToken,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to merge auth.json: %w", err)
	}

	var backupFile string
	if authExists && shouldBackupAuthJSON && len(bytes.TrimSpace(authExisting)) > 0 {
		backupFile, err = createBackupFile(authJSONPath, authExisting, authPerm)
		if err != nil {
			return nil, fmt.Errorf("failed to backup broken auth.json: %w", err)
		}
	}

	authUpdated := !authExists || !bytes.Equal(authExisting, authMerged)
	if authUpdated {
		if !authExists {
			authPerm = 0600
		}
		if err := writeFileAtomically(authJSONPath, authMerged, authPerm); err != nil {
			return nil, fmt.Errorf("failed to write auth.json: %w", err)
		}
	}

	return &CodexLocalConfigSyncResult{
		Success: true,
		BaseURL: baseURL,
		WrittenFiles: []string{
			configTomlPath,
			authJSONPath,
		},
		RecoveredAuthJSON: recoveredAuthJSON,
		BackupFile:        backupFile,
		Details: CodexLocalConfigSyncDetails{
			ConfigTomlUpdated: configUpdated,
			AuthJSONUpdated:   authUpdated,
		},
		Message: "codex local configuration synced",
	}, nil
}

func mergeCodexConfigTOML(existing []byte, providerName string, baseURL string, model string) ([]byte, error) {
	doc := map[string]any{}
	if len(bytes.TrimSpace(existing)) > 0 {
		if err := toml.Unmarshal(existing, &doc); err != nil {
			return nil, fmt.Errorf("invalid TOML: %w", err)
		}
	}

	doc["model_provider"] = providerName

	modelProviders := getNestedMap(doc, "model_providers")
	providerConfig := getNestedMap(modelProviders, providerName)
	providerConfig["name"] = providerName
	providerConfig["base_url"] = baseURL
	providerConfig["wire_api"] = "responses"
	providerConfig["request_max_retries"] = 4
	providerConfig["stream_max_retries"] = 10
	providerConfig["stream_idle_timeout_ms"] = 300000
	if strings.TrimSpace(model) != "" {
		providerConfig["model"] = strings.TrimSpace(model)
	}

	modelProviders[providerName] = providerConfig
	doc["model_providers"] = modelProviders

	out, err := toml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal TOML: %w", err)
	}

	return out, nil
}

func mergeCodexAuthJSON(
	existing []byte,
	providerName string,
	apiToken string,
) ([]byte, bool, bool, error) {
	doc := map[string]any{}
	recoveredAuthJSON := false
	shouldBackup := false
	if len(bytes.TrimSpace(existing)) > 0 {
		if err := json.Unmarshal(existing, &doc); err != nil {
			recoveredAuthJSON = true
			sanitized := sanitizeBrokenJSON(existing)
			if sanitizeErr := json.Unmarshal(sanitized, &doc); sanitizeErr != nil {
				shouldBackup = true
				doc = map[string]any{}
			}
		}
	}

	trimmedProvider := strings.TrimSpace(providerName)
	candidateKeys := []string{"maxx"}
	if trimmedProvider != "" && trimmedProvider != "maxx" {
		candidateKeys = append([]string{trimmedProvider}, candidateKeys...)
	}

	// Migrate legacy nested auth format: {"maxx":{"OPENAI_API_KEY":"..."}}
	for _, key := range candidateKeys {
		raw, exists := doc[key]
		if !exists {
			continue
		}
		nested, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if _, hasToken := nested["OPENAI_API_KEY"]; hasToken {
			delete(nested, "OPENAI_API_KEY")
			if len(nested) == 0 {
				delete(doc, key)
			} else {
				doc[key] = nested
			}
		}
	}

	// Current target format is flat auth.json
	doc["OPENAI_API_KEY"] = apiToken

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, recoveredAuthJSON, shouldBackup, fmt.Errorf("marshal JSON: %w", err)
	}

	out = append(out, '\n')
	return out, recoveredAuthJSON, shouldBackup, nil
}

func getNestedMap(root map[string]any, key string) map[string]any {
	if raw, exists := root[key]; exists {
		if m, ok := raw.(map[string]any); ok {
			return m
		}
	}
	newMap := map[string]any{}
	root[key] = newMap
	return newMap
}

func readFileIfExists(path string) ([]byte, bool, os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, 0, nil
		}
		return nil, false, 0, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, true, info.Mode().Perm(), err
	}

	return data, true, info.Mode().Perm(), nil
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(perm); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	// On some Windows setups os.Rename cannot replace existing files.
	if err := os.Rename(tmpPath, path); err != nil {
		if writeErr := os.WriteFile(path, data, perm); writeErr != nil {
			return fmt.Errorf("rename failed: %v, fallback write failed: %w", err, writeErr)
		}
	}

	return nil
}

func createBackupFile(path string, data []byte, perm os.FileMode) (string, error) {
	backupPath := buildBackupPath(path)
	if perm == 0 {
		perm = 0600
	}
	if err := writeFileAtomically(backupPath, data, perm); err != nil {
		return "", err
	}
	return backupPath, nil
}

func buildBackupPath(path string) string {
	base := path + ".bak-" + time.Now().Format("20060102-150405")
	candidate := base
	for i := 1; ; i++ {
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

var trailingCommaRE = regexp.MustCompile(`,\s*([}\]])`)

func sanitizeBrokenJSON(input []byte) []byte {
	trimmed := bytes.TrimSpace(input)
	trimmed = bytes.TrimPrefix(trimmed, []byte{0xEF, 0xBB, 0xBF})
	cleaned := trailingCommaRE.ReplaceAll(trimmed, []byte("$1"))
	return bytes.TrimSpace(cleaned)
}

func deriveRequestBaseURL(r *http.Request, fallbackAddr string) string {
	host := firstCSVHeaderValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	host = sanitizeHost(host)

	if host == "" {
		host = fallbackHost(fallbackAddr)
	}

	proto := strings.ToLower(firstCSVHeaderValue(r.Header.Get("X-Forwarded-Proto")))
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	if proto != "https" {
		proto = "http"
	}

	return proto + "://" + host
}

func firstCSVHeaderValue(raw string) string {
	if raw == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(raw, ",")[0])
}

func sanitizeHost(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "https://")
	value = strings.Split(value, "/")[0]
	return strings.TrimSpace(value)
}

func fallbackHost(serverAddr string) string {
	addr := strings.TrimSpace(serverAddr)
	if addr == "" {
		return "localhost:9880"
	}

	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}

	if strings.Contains(addr, ":") {
		return addr
	}

	if _, err := strconv.Atoi(addr); err == nil {
		return "localhost:" + addr
	}

	return addr
}

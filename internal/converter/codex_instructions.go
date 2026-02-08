package converter

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync/atomic"
)

//go:embed codex_instructions/default.md
var defaultPrompt string

//go:embed codex_instructions/codex.md
var codexPrompt string

//go:embed codex_instructions/codex_max.md
var codexMaxPrompt string

//go:embed codex_instructions/gpt51.md
var gpt51Prompt string

//go:embed codex_instructions/gpt52.md
var gpt52Prompt string

//go:embed codex_instructions/gpt53.md
var gpt53Prompt string

//go:embed codex_instructions/gpt52_codex.md
var gpt52CodexPrompt string

//go:embed opencode_codex_instructions.txt
var opencodeCodexInstructions string

const (
	codexUserAgentKey  = "__cpa_user_agent"
	userAgentOpenAISDK = "opencode/"
)

var codexInstructionsEnabled atomic.Bool

// SetCodexInstructionsEnabled sets whether codex instructions processing is enabled.
func SetCodexInstructionsEnabled(enabled bool) {
	codexInstructionsEnabled.Store(enabled)
}

// GetCodexInstructionsEnabled returns whether codex instructions processing is enabled.
func GetCodexInstructionsEnabled() bool {
	if settings := GetGlobalSettings(); settings != nil {
		return settings.CodexInstructionsEnabled
	}
	return codexInstructionsEnabled.Load()
}

// InjectCodexUserAgent injects user agent into a request body for codex instruction selection.
func InjectCodexUserAgent(raw []byte, userAgent string) []byte {
	if len(raw) == 0 {
		return raw
	}
	trimmed := strings.TrimSpace(userAgent)
	if trimmed == "" {
		return raw
	}
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return raw
	}
	data[codexUserAgentKey] = trimmed
	return mustMarshal(data)
}

// ExtractCodexUserAgent extracts the user agent from a request body.
func ExtractCodexUserAgent(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	if v, ok := data[codexUserAgentKey].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// StripCodexUserAgent removes the injected user agent from the body.
func StripCodexUserAgent(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return raw
	}
	if _, ok := data[codexUserAgentKey]; !ok {
		return raw
	}
	delete(data, codexUserAgentKey)
	return mustMarshal(data)
}

func useOpenCodeInstructions(userAgent string) bool {
	return strings.Contains(strings.ToLower(userAgent), userAgentOpenAISDK)
}

func codexInstructionsForCodex(modelName string) string {
	switch {
	case strings.Contains(modelName, "codex-max"):
		return codexMaxPrompt
	case strings.Contains(modelName, "5.2-codex"):
		return gpt52CodexPrompt
	case strings.Contains(modelName, "5.3-codex"):
		return gpt52CodexPrompt
	case strings.Contains(modelName, "codex"):
		return codexPrompt
	case strings.Contains(modelName, "5.1"):
		return gpt51Prompt
	case strings.Contains(modelName, "5.2"):
		return gpt52Prompt
	case strings.Contains(modelName, "5.3"):
		return gpt53Prompt
	default:
		return defaultPrompt
	}
}

// CodexInstructionsForModel returns official instructions based on model and user agent.
func CodexInstructionsForModel(modelName, userAgent string) string {
	if !GetCodexInstructionsEnabled() {
		return ""
	}
	if useOpenCodeInstructions(userAgent) {
		return opencodeCodexInstructions
	}
	return codexInstructionsForCodex(modelName)
}

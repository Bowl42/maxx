package antigravity

import (
	"log"
	"strings"
)

// ModelMappingRule represents a single model mapping rule
// Rules are matched in order, first match wins
type ModelMappingRule struct {
	Pattern string // Source pattern, supports * wildcard
	Target  string // Target model name
}

// defaultModelMappingRules is the ordered list of default mapping rules
// Rules are matched in order, first match wins (higher priority first)
// Supports wildcard patterns: * matches any characters
// Note: gemini-* models pass through automatically without needing a mapping rule
var defaultModelMappingRules = []ModelMappingRule{
	// OpenAI 协议映射表 - 按优先级排序
	{"gpt-4o-mini*", "gemini-2.5-flash"}, // gpt-4o-mini 系列 (优先于 gpt-4o)
	{"gpt-4o*", "gemini-3-flash"},        // gpt-4o 系列 (优先于 gpt-4)
	{"gpt-4*", "gemini-3-pro-high"},      // 所有 gpt-4 变体
	{"gpt-3.5*", "gemini-2.5-flash"},     // 所有 gpt-3.5 变体
	{"o1-*", "gemini-3-pro-high"},        // OpenAI o1 系列
	{"o3-*", "gemini-3-pro-high"},        // OpenAI o3 系列

	// Claude 模型 - 具体模式优先
	{"claude-3-5-sonnet-*", "claude-sonnet-4-5"},    // Claude 3.5 Sonnet
	{"claude-3-opus-*", "claude-opus-4-6-thinking"}, // Claude 3 Opus
	{"claude-opus-4-6*", "claude-opus-4-6-thinking"}, // Claude Opus 4.6 通配符
	{"claude-opus-4-5*", "claude-opus-4-5-thinking"}, // Claude Opus 4.5 通配符
	{"claude-opus-4-*", "claude-opus-4-6-thinking"}, // Claude 4 Opus
	{"claude-haiku-*", "gemini-2.5-flash-lite"},     // Claude Haiku
	{"claude-3-haiku-*", "gemini-2.5-flash-lite"},   // Claude 3 Haiku

	// 通用 Claude 回退 (宽泛通配符放最后)
	{"*opus*", "claude-opus-4-6-thinking"}, // 所有 opus 变体
	{"*sonnet*", "claude-sonnet-4-5"},      // 所有 sonnet 变体
	{"*haiku*", "gemini-2.5-flash-lite"},   // 所有 haiku 变体
}

// GetDefaultModelMapping returns the default mapping as a map (for API compatibility)
// Note: The map loses ordering, use defaultModelMappingRules for ordered matching
func GetDefaultModelMapping() map[string]string {
	result := make(map[string]string, len(defaultModelMappingRules))
	for _, rule := range defaultModelMappingRules {
		result[rule.Pattern] = rule.Target
	}
	return result
}

// AvailableTargetModels is the list of valid target models for mapping
var AvailableTargetModels = []string{
	// Claude models
	"claude-opus-4-6-thinking",
	"claude-opus-4-5-thinking",
	"claude-sonnet-4-5",
	"claude-sonnet-4-5-thinking",
	// Gemini models
	"gemini-2.5-flash-lite",
	"gemini-2.5-flash",
	"gemini-2.5-flash-thinking",
	"gemini-2.5-pro",
	"gemini-3-flash",
	"gemini-3-pro",
	"gemini-3-pro-low",
	"gemini-3-pro-high",
	"gemini-3-pro-preview",
	"gemini-3-pro-image",
}

// GetAvailableTargetModels returns the list of valid target models
func GetAvailableTargetModels() []string {
	return AvailableTargetModels
}

// MapClaudeModelToGemini maps Claude model names to Gemini model names
// DEPRECATED: Model mapping is now handled centrally in executor.mapModel().
// This function is kept for backward compatibility but should not be used.
// It simply returns the input as-is since mapping is done elsewhere.
func MapClaudeModelToGemini(input string) string {
	return MapClaudeModelToGeminiWithConfig(input, "")
}

// MapClaudeModelToGeminiWithConfig maps Claude model names with optional haikuTarget override
// DEPRECATED: Model mapping is now handled centrally in executor.mapModel().
// This function only handles haikuTarget override and pass-through logic.
// The actual mapping rules (global settings, default rules) are applied in executor.mapModel().
func MapClaudeModelToGeminiWithConfig(input string, haikuTarget string) string {
	// Strip -online suffix for mapping lookup (will be re-added by resolveRequestConfig)
	cleanInput := strings.TrimSuffix(input, "-online")

	// 1. Check if this is a Haiku model and apply haikuTarget override
	// This is the only mapping logic that should remain here, as it's provider-specific
	if haikuTarget != "" && isHaikuModel(cleanInput) {
		return haikuTarget
	}

	// 2. Pass-through: model mapping is now handled in executor.mapModel()
	// Just return the input as-is
	return cleanInput
}

// MatchRulesInOrder matches input against rules in order, first match wins
// Returns the target model or empty string if no match
func MatchRulesInOrder(input string, rules []ModelMappingRule) string {
	for i, rule := range rules {
		matched := MatchWildcard(rule.Pattern, input)
		log.Printf("[MatchRulesInOrder] Rule[%d]: pattern=%q, input=%q, matched=%v", i, rule.Pattern, input, matched)
		if matched {
			log.Printf("[MatchRulesInOrder] Matched! Returning target=%q", rule.Target)
			return rule.Target
		}
	}
	return ""
}

// MatchWildcard checks if input matches a wildcard pattern
// Supports * as wildcard matching any characters
// Examples:
//   - "claude-3-5-sonnet-*" matches "claude-3-5-sonnet-20241022"
//   - "*haiku*" matches "claude-haiku-4", "claude-3-haiku-20240307"
//   - "gpt-4-*" matches "gpt-4-turbo", "gpt-4-0613"
func MatchWildcard(pattern, input string) bool {
	// Simple cases
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == input
	}

	parts := strings.Split(pattern, "*")

	// Handle prefix-only pattern: "prefix*"
	if len(parts) == 2 && parts[1] == "" {
		return strings.HasPrefix(input, parts[0])
	}

	// Handle suffix-only pattern: "*suffix"
	if len(parts) == 2 && parts[0] == "" {
		return strings.HasSuffix(input, parts[1])
	}

	// Handle patterns with multiple wildcards
	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}

		idx := strings.Index(input[pos:], part)
		if idx < 0 {
			return false
		}

		// First part must be at the beginning if pattern doesn't start with *
		if i == 0 && idx != 0 {
			return false
		}

		pos += idx + len(part)
	}

	// Last part must be at the end if pattern doesn't end with *
	if parts[len(parts)-1] != "" && !strings.HasSuffix(input, parts[len(parts)-1]) {
		return false
	}

	return true
}

// isHaikuModel checks if the model name is a Haiku variant
func isHaikuModel(model string) bool {
	modelLower := strings.ToLower(model)
	return strings.Contains(modelLower, "haiku")
}

// ParseImageConfig parses image configuration from model name suffixes
// Returns imageConfig and cleanModelName
func ParseImageConfig(modelName string) (map[string]interface{}, string) {
	aspectRatio := "1:1"

	switch {
	case strings.Contains(modelName, "-21x9") || strings.Contains(modelName, "-21-9"):
		aspectRatio = "21:9"
	case strings.Contains(modelName, "-16x9") || strings.Contains(modelName, "-16-9"):
		aspectRatio = "16:9"
	case strings.Contains(modelName, "-9x16") || strings.Contains(modelName, "-9-16"):
		aspectRatio = "9:16"
	case strings.Contains(modelName, "-4x3") || strings.Contains(modelName, "-4-3"):
		aspectRatio = "4:3"
	case strings.Contains(modelName, "-3x4") || strings.Contains(modelName, "-3-4"):
		aspectRatio = "3:4"
	case strings.Contains(modelName, "-1x1") || strings.Contains(modelName, "-1-1"):
		aspectRatio = "1:1"
	}

	isHD := strings.Contains(modelName, "-4k") || strings.Contains(modelName, "-hd")
	is2K := strings.Contains(modelName, "-2k")

	config := map[string]interface{}{
		"aspectRatio": aspectRatio,
	}

	if isHD {
		config["imageSize"] = "4K"
	} else if is2K {
		config["imageSize"] = "2K"
	}

	// The upstream model must be EXACTLY "gemini-3-pro-image"
	return config, "gemini-3-pro-image"
}

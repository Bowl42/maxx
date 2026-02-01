package custom

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/awsl-project/maxx/internal/domain"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Claude Code system prompt for cloaking
const claudeCodeSystemPrompt = `You are Claude Code, Anthropic's official CLI for Claude.`

const claudeToolPrefix = "proxy_"

// userIDPattern matches Claude Code format: user_[64-hex]_account__session_[uuid-v4]
var userIDPattern = regexp.MustCompile(`^user_[a-fA-F0-9]{64}_account__session_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// processClaudeRequestBody processes Claude request body before sending to upstream.
// Following CLIProxyAPI order:
// 1. applyCloaking (system prompt injection, fake user_id, sensitive word obfuscation)
// 2. disableThinkingIfToolChoiceForced
// 3. ensureCacheControl (auto-inject if missing)
// 4. extractAndRemoveBetas
// Returns processed body and extra betas for header.
func processClaudeRequestBody(body []byte, clientUserAgent string, cloakCfg *domain.ProviderConfigCustomCloak) ([]byte, []string) {
	modelName := gjson.GetBytes(body, "model").String()

	// 1. Apply cloaking (system prompt injection, fake user_id, sensitive word obfuscation)
	body = applyCloaking(body, clientUserAgent, modelName, cloakCfg)

	// 2. Disable thinking if tool_choice forces tool use
	body = disableThinkingIfToolChoiceForced(body)

	// 3. Ensure minimum thinking budget if present
	body = ensureMinThinkingBudget(body)

	// 4. Auto-inject cache_control if missing (CLIProxyAPI behavior)
	if countCacheControls(body) == 0 {
		body = ensureCacheControl(body)
	}

	// 5. Extract betas from body (to be added to header)
	var extraBetas []string
	extraBetas, body = extractAndRemoveBetas(body)

	return body, extraBetas
}

// applyCloaking applies cloaking transformations based on config and client.
// Cloaking includes: system prompt injection, fake user ID, sensitive word obfuscation.
func applyCloaking(body []byte, clientUserAgent string, model string, cloakCfg *domain.ProviderConfigCustomCloak) []byte {
	var cloakMode string
	var strictMode bool
	var sensitiveWords []string

	if cloakCfg != nil {
		cloakMode = strings.TrimSpace(cloakCfg.Mode)
		strictMode = cloakCfg.StrictMode
		sensitiveWords = cloakCfg.SensitiveWords
	}

	// Default mode is "auto"
	if !shouldCloak(cloakMode, clientUserAgent) {
		return body
	}

	// Skip system instructions for claude-3-5-haiku models (CLIProxyAPI behavior)
	if !strings.HasPrefix(model, "claude-3-5-haiku") {
		body = checkSystemInstructionsWithMode(body, strictMode)
	}

	// Inject fake user_id
	body = injectFakeUserID(body)

	// Apply sensitive word obfuscation
	if len(sensitiveWords) > 0 {
		matcher := buildSensitiveWordMatcher(sensitiveWords)
		body = obfuscateSensitiveWords(body, matcher)
	}

	return body
}

// isClaudeCodeClient checks if the User-Agent indicates a Claude Code client.
func isClaudeCodeClient(userAgent string) bool {
	return strings.HasPrefix(userAgent, "claude-cli")
}

func isClaudeOAuthToken(apiKey string) bool {
	return strings.Contains(apiKey, "sk-ant-oat")
}

func ensureMinThinkingBudget(body []byte) []byte {
	const minBudget = 1024
	result := gjson.GetBytes(body, "thinking.enabled.budget_tokens")
	if result.Type != gjson.Number {
		return body
	}
	if result.Int() >= minBudget {
		return body
	}
	updated, err := sjson.SetBytes(body, "thinking.enabled.budget_tokens", minBudget)
	if err != nil {
		return body
	}
	return updated
}

func applyClaudeToolPrefix(body []byte, prefix string) []byte {
	if prefix == "" {
		return body
	}

	if tools := gjson.GetBytes(body, "tools"); tools.Exists() && tools.IsArray() {
		tools.ForEach(func(index, tool gjson.Result) bool {
			// Skip built-in tools (web_search, code_execution, etc.) which have
			// a "type" field and require their name to remain unchanged.
			if tool.Get("type").Exists() && tool.Get("type").String() != "" {
				return true
			}
			name := tool.Get("name").String()
			if name == "" || strings.HasPrefix(name, prefix) {
				return true
			}
			path := fmt.Sprintf("tools.%d.name", index.Int())
			body, _ = sjson.SetBytes(body, path, prefix+name)
			return true
		})
	}

	if gjson.GetBytes(body, "tool_choice.type").String() == "tool" {
		name := gjson.GetBytes(body, "tool_choice.name").String()
		if name != "" && !strings.HasPrefix(name, prefix) {
			body, _ = sjson.SetBytes(body, "tool_choice.name", prefix+name)
		}
	}

	if messages := gjson.GetBytes(body, "messages"); messages.Exists() && messages.IsArray() {
		messages.ForEach(func(msgIndex, msg gjson.Result) bool {
			content := msg.Get("content")
			if !content.Exists() || !content.IsArray() {
				return true
			}
			content.ForEach(func(contentIndex, part gjson.Result) bool {
				if part.Get("type").String() != "tool_use" {
					return true
				}
				name := part.Get("name").String()
				if name == "" || strings.HasPrefix(name, prefix) {
					return true
				}
				path := fmt.Sprintf("messages.%d.content.%d.name", msgIndex.Int(), contentIndex.Int())
				body, _ = sjson.SetBytes(body, path, prefix+name)
				return true
			})
			return true
		})
	}

	return body
}

func stripClaudeToolPrefixFromResponse(body []byte, prefix string) []byte {
	if prefix == "" {
		return body
	}
	content := gjson.GetBytes(body, "content")
	if !content.Exists() || !content.IsArray() {
		return body
	}
	content.ForEach(func(index, part gjson.Result) bool {
		if part.Get("type").String() != "tool_use" {
			return true
		}
		name := part.Get("name").String()
		if !strings.HasPrefix(name, prefix) {
			return true
		}
		path := fmt.Sprintf("content.%d.name", index.Int())
		body, _ = sjson.SetBytes(body, path, strings.TrimPrefix(name, prefix))
		return true
	})
	return body
}

func stripClaudeToolPrefixFromStreamLine(line []byte, prefix string) []byte {
	if prefix == "" {
		return line
	}
	payload := jsonPayload(line)
	if len(payload) == 0 || !gjson.ValidBytes(payload) {
		return line
	}
	contentBlock := gjson.GetBytes(payload, "content_block")
	if !contentBlock.Exists() || contentBlock.Get("type").String() != "tool_use" {
		return line
	}
	name := contentBlock.Get("name").String()
	if !strings.HasPrefix(name, prefix) {
		return line
	}
	updated, err := sjson.SetBytes(payload, "content_block.name", strings.TrimPrefix(name, prefix))
	if err != nil {
		return line
	}

	trimmed := bytes.TrimSpace(line)
	if bytes.HasPrefix(trimmed, []byte("data:")) {
		return append([]byte("data: "), updated...)
	}
	return updated
}

func jsonPayload(line []byte) []byte {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return nil
	}
	if bytes.Equal(trimmed, []byte("[DONE]")) {
		return nil
	}
	if bytes.HasPrefix(trimmed, []byte("event:")) {
		return nil
	}
	if bytes.HasPrefix(trimmed, []byte("data:")) {
		trimmed = bytes.TrimSpace(trimmed[len("data:"):])
	}
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil
	}
	return trimmed
}

// injectClaudeCodeSystemPrompt injects Claude Code system prompt into the request.
// This is the non-strict cloaking behavior (prepend prompt).
func injectClaudeCodeSystemPrompt(body []byte) []byte {
	return checkSystemInstructionsWithMode(body, false)
}

// injectFakeUserID generates and injects a fake user_id into the request metadata.
// Only injects if user_id is missing or invalid.
func injectFakeUserID(body []byte) []byte {
	existingUserID := gjson.GetBytes(body, "metadata.user_id").String()
	if existingUserID != "" && isValidUserID(existingUserID) {
		return body
	}

	// Generate and inject fake user_id
	body, _ = sjson.SetBytes(body, "metadata.user_id", generateFakeUserID())
	return body
}

// shouldCloak determines if request should be cloaked based on config and client User-Agent.
// Returns true if cloaking should be applied.
func shouldCloak(cloakMode string, userAgent string) bool {
	switch strings.ToLower(strings.TrimSpace(cloakMode)) {
	case "always":
		return true
	case "never":
		return false
	default: // "auto" or empty
		return !strings.HasPrefix(userAgent, "claude-cli")
	}
}

// isValidUserID checks if a user_id matches Claude Code format.
func isValidUserID(userID string) bool {
	return userIDPattern.MatchString(userID)
}

// generateFakeUserID generates a fake user_id in Claude Code format.
// Format: user_{64-hex}_account__session_{uuid}
func generateFakeUserID() string {
	// Generate 32 random bytes (64 hex chars)
	randomBytes := make([]byte, 32)
	_, _ = rand.Read(randomBytes)
	hexPart := hex.EncodeToString(randomBytes)

	// Generate UUID for session
	sessionUUID := uuid.New().String()

	return "user_" + hexPart + "_account__session_" + sessionUUID
}

// disableThinkingIfToolChoiceForced checks if tool_choice forces tool use and disables thinking.
// Anthropic API does not allow thinking when tool_choice is set to "any" or "tool".
// See: https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking#important-considerations
func disableThinkingIfToolChoiceForced(body []byte) []byte {
	toolChoiceType := gjson.GetBytes(body, "tool_choice.type").String()
	// "auto" is allowed with thinking, but "any" or "tool" (specific tool) are not
	if toolChoiceType == "any" || toolChoiceType == "tool" {
		// Remove thinking configuration entirely to avoid API error
		body, _ = sjson.DeleteBytes(body, "thinking")
	}
	return body
}

// extractAndRemoveBetas extracts betas array from request body and removes it.
// Returns the extracted betas and the modified body.
func extractAndRemoveBetas(body []byte) ([]string, []byte) {
	betasResult := gjson.GetBytes(body, "betas")
	if !betasResult.Exists() {
		return nil, body
	}

	var betas []string
	if betasResult.IsArray() {
		for _, item := range betasResult.Array() {
			if s := strings.TrimSpace(item.String()); s != "" {
				betas = append(betas, s)
			}
		}
	} else if s := strings.TrimSpace(betasResult.String()); s != "" {
		betas = append(betas, s)
	}

	body, _ = sjson.DeleteBytes(body, "betas")
	return betas, body
}

// checkSystemInstructionsWithMode injects Claude Code system prompt.
// In strict mode, it replaces all user system messages.
// In non-strict mode (default), it prepends to existing system messages.
func checkSystemInstructionsWithMode(body []byte, strictMode bool) []byte {
	system := gjson.GetBytes(body, "system")
	claudeCodeInstructions := `[{"type":"text","text":"` + claudeCodeSystemPrompt + `"}]`

	if strictMode {
		body, _ = sjson.SetRawBytes(body, "system", []byte(claudeCodeInstructions))
		return body
	}

	if system.IsArray() {
		if gjson.GetBytes(body, "system.0.text").String() != claudeCodeSystemPrompt {
			system.ForEach(func(_, part gjson.Result) bool {
				if part.Get("type").String() == "text" {
					claudeCodeInstructions, _ = sjson.SetRaw(claudeCodeInstructions, "-1", part.Raw)
				}
				return true
			})
			body, _ = sjson.SetRawBytes(body, "system", []byte(claudeCodeInstructions))
		}
	} else {
		body, _ = sjson.SetRawBytes(body, "system", []byte(claudeCodeInstructions))
	}
	return body
}

// ===== Sensitive word obfuscation (CLIProxyAPI-aligned) =====

// zeroWidthSpace is the Unicode zero-width space character used for obfuscation.
const zeroWidthSpace = "\u200B"

// SensitiveWordMatcher holds the compiled regex for matching sensitive words.
type SensitiveWordMatcher struct {
	regex *regexp.Regexp
}

// buildSensitiveWordMatcher compiles a regex from the word list.
// Words are sorted by length (longest first) for proper matching.
func buildSensitiveWordMatcher(words []string) *SensitiveWordMatcher {
	if len(words) == 0 {
		return nil
	}

	var validWords []string
	for _, w := range words {
		w = strings.TrimSpace(w)
		if utf8.RuneCountInString(w) >= 2 && !strings.Contains(w, zeroWidthSpace) {
			validWords = append(validWords, w)
		}
	}
	if len(validWords) == 0 {
		return nil
	}

	sort.Slice(validWords, func(i, j int) bool {
		return len(validWords[i]) > len(validWords[j])
	})

	escaped := make([]string, len(validWords))
	for i, w := range validWords {
		escaped[i] = regexp.QuoteMeta(w)
	}

	pattern := "(?i)" + strings.Join(escaped, "|")
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	return &SensitiveWordMatcher{regex: re}
}

// obfuscateWord inserts a zero-width space after the first grapheme.
func obfuscateWord(word string) string {
	if strings.Contains(word, zeroWidthSpace) {
		return word
	}
	r, size := utf8.DecodeRuneInString(word)
	if r == utf8.RuneError || size >= len(word) {
		return word
	}
	return string(r) + zeroWidthSpace + word[size:]
}

// obfuscateText replaces all sensitive words in the text.
func (m *SensitiveWordMatcher) obfuscateText(text string) string {
	if m == nil || m.regex == nil {
		return text
	}
	return m.regex.ReplaceAllStringFunc(text, obfuscateWord)
}

// obfuscateSensitiveWords processes the payload and obfuscates sensitive words
// in system blocks and message content.
func obfuscateSensitiveWords(payload []byte, matcher *SensitiveWordMatcher) []byte {
	if matcher == nil || matcher.regex == nil {
		return payload
	}
	payload = obfuscateSystemBlocks(payload, matcher)
	payload = obfuscateMessages(payload, matcher)
	return payload
}

// obfuscateSystemBlocks obfuscates sensitive words in system blocks.
func obfuscateSystemBlocks(payload []byte, matcher *SensitiveWordMatcher) []byte {
	system := gjson.GetBytes(payload, "system")
	if !system.Exists() {
		return payload
	}

	if system.IsArray() {
		modified := false
		system.ForEach(func(key, value gjson.Result) bool {
			if value.Get("type").String() == "text" {
				text := value.Get("text").String()
				obfuscated := matcher.obfuscateText(text)
				if obfuscated != text {
					path := "system." + key.String() + ".text"
					payload, _ = sjson.SetBytes(payload, path, obfuscated)
					modified = true
				}
			}
			return true
		})
		if modified {
			return payload
		}
	} else if system.Type == gjson.String {
		text := system.String()
		obfuscated := matcher.obfuscateText(text)
		if obfuscated != text {
			payload, _ = sjson.SetBytes(payload, "system", obfuscated)
		}
	}

	return payload
}

// obfuscateMessages obfuscates sensitive words in message content.
func obfuscateMessages(payload []byte, matcher *SensitiveWordMatcher) []byte {
	messages := gjson.GetBytes(payload, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return payload
	}

	messages.ForEach(func(msgKey, msg gjson.Result) bool {
		content := msg.Get("content")
		if !content.Exists() {
			return true
		}

		msgPath := "messages." + msgKey.String()

		if content.Type == gjson.String {
			text := content.String()
			obfuscated := matcher.obfuscateText(text)
			if obfuscated != text {
				payload, _ = sjson.SetBytes(payload, msgPath+".content", obfuscated)
			}
		} else if content.IsArray() {
			content.ForEach(func(blockKey, block gjson.Result) bool {
				if block.Get("type").String() == "text" {
					text := block.Get("text").String()
					obfuscated := matcher.obfuscateText(text)
					if obfuscated != text {
						path := msgPath + ".content." + blockKey.String() + ".text"
						payload, _ = sjson.SetBytes(payload, path, obfuscated)
					}
				}
				return true
			})
		}

		return true
	})

	return payload
}

// ===== Cache control injection (CLIProxyAPI-aligned) =====

// ensureCacheControl injects cache_control breakpoints into the payload for optimal prompt caching.
// According to Anthropic's documentation, cache prefixes are created in order: tools -> system -> messages.
func ensureCacheControl(payload []byte) []byte {
	payload = injectToolsCacheControl(payload)
	payload = injectSystemCacheControl(payload)
	payload = injectMessagesCacheControl(payload)
	return payload
}

func countCacheControls(payload []byte) int {
	count := 0

	system := gjson.GetBytes(payload, "system")
	if system.IsArray() {
		system.ForEach(func(_, item gjson.Result) bool {
			if item.Get("cache_control").Exists() {
				count++
			}
			return true
		})
	}

	tools := gjson.GetBytes(payload, "tools")
	if tools.IsArray() {
		tools.ForEach(func(_, item gjson.Result) bool {
			if item.Get("cache_control").Exists() {
				count++
			}
			return true
		})
	}

	messages := gjson.GetBytes(payload, "messages")
	if messages.IsArray() {
		messages.ForEach(func(_, msg gjson.Result) bool {
			content := msg.Get("content")
			if content.IsArray() {
				content.ForEach(func(_, item gjson.Result) bool {
					if item.Get("cache_control").Exists() {
						count++
					}
					return true
				})
			}
			return true
		})
	}

	return count
}

// injectMessagesCacheControl adds cache_control to the second-to-last user turn for multi-turn caching.
func injectMessagesCacheControl(payload []byte) []byte {
	messages := gjson.GetBytes(payload, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return payload
	}

	hasCacheControlInMessages := false
	messages.ForEach(func(_, msg gjson.Result) bool {
		content := msg.Get("content")
		if content.IsArray() {
			content.ForEach(func(_, item gjson.Result) bool {
				if item.Get("cache_control").Exists() {
					hasCacheControlInMessages = true
					return false
				}
				return true
			})
		}
		return !hasCacheControlInMessages
	})
	if hasCacheControlInMessages {
		return payload
	}

	var userMsgIndices []int
	messages.ForEach(func(index gjson.Result, msg gjson.Result) bool {
		if msg.Get("role").String() == "user" {
			userMsgIndices = append(userMsgIndices, int(index.Int()))
		}
		return true
	})
	if len(userMsgIndices) < 2 {
		return payload
	}

	secondToLastUserIdx := userMsgIndices[len(userMsgIndices)-2]
	contentPath := fmt.Sprintf("messages.%d.content", secondToLastUserIdx)
	content := gjson.GetBytes(payload, contentPath)

	if content.IsArray() {
		contentCount := int(content.Get("#").Int())
		if contentCount > 0 {
			cacheControlPath := fmt.Sprintf("messages.%d.content.%d.cache_control", secondToLastUserIdx, contentCount-1)
			result, err := sjson.SetBytes(payload, cacheControlPath, map[string]string{"type": "ephemeral"})
			if err != nil {
				log.Printf("failed to inject cache_control into messages: %v", err)
				return payload
			}
			payload = result
		}
	} else if content.Type == gjson.String {
		text := content.String()
		newContent := []map[string]interface{}{
			{
				"type": "text",
				"text": text,
				"cache_control": map[string]string{
					"type": "ephemeral",
				},
			},
		}
		result, err := sjson.SetBytes(payload, contentPath, newContent)
		if err != nil {
			log.Printf("failed to inject cache_control into message string content: %v", err)
			return payload
		}
		payload = result
	}

	return payload
}

// injectToolsCacheControl adds cache_control to the last tool in the tools array.
func injectToolsCacheControl(payload []byte) []byte {
	tools := gjson.GetBytes(payload, "tools")
	if !tools.Exists() || !tools.IsArray() {
		return payload
	}

	toolCount := int(tools.Get("#").Int())
	if toolCount == 0 {
		return payload
	}

	hasCacheControlInTools := false
	tools.ForEach(func(_, tool gjson.Result) bool {
		if tool.Get("cache_control").Exists() {
			hasCacheControlInTools = true
			return false
		}
		return true
	})
	if hasCacheControlInTools {
		return payload
	}

	lastToolPath := fmt.Sprintf("tools.%d.cache_control", toolCount-1)
	result, err := sjson.SetBytes(payload, lastToolPath, map[string]string{"type": "ephemeral"})
	if err != nil {
		log.Printf("failed to inject cache_control into tools array: %v", err)
		return payload
	}

	return result
}

// injectSystemCacheControl adds cache_control to the last element in the system prompt.
func injectSystemCacheControl(payload []byte) []byte {
	system := gjson.GetBytes(payload, "system")
	if !system.Exists() {
		return payload
	}

	if system.IsArray() {
		count := int(system.Get("#").Int())
		if count == 0 {
			return payload
		}

		hasCacheControlInSystem := false
		system.ForEach(func(_, item gjson.Result) bool {
			if item.Get("cache_control").Exists() {
				hasCacheControlInSystem = true
				return false
			}
			return true
		})
		if hasCacheControlInSystem {
			return payload
		}

		lastSystemPath := fmt.Sprintf("system.%d.cache_control", count-1)
		result, err := sjson.SetBytes(payload, lastSystemPath, map[string]string{"type": "ephemeral"})
		if err != nil {
			log.Printf("failed to inject cache_control into system array: %v", err)
			return payload
		}
		payload = result
	} else if system.Type == gjson.String {
		text := system.String()
		newSystem := []map[string]interface{}{
			{
				"type": "text",
				"text": text,
				"cache_control": map[string]string{
					"type": "ephemeral",
				},
			},
		}
		result, err := sjson.SetBytes(payload, "system", newSystem)
		if err != nil {
			log.Printf("failed to inject cache_control into system string: %v", err)
			return payload
		}
		payload = result
	}

	return payload
}

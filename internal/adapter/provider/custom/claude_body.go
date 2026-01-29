package custom

import (
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// processClaudeRequestBody processes Claude request body before sending to upstream.
// It extracts betas to header and handles thinking/tool_choice constraints.
// Returns the processed body.
func processClaudeRequestBody(body []byte, req *http.Request) []byte {
	// 1. Extract betas from body and merge to Anthropic-Beta header
	var betas []string
	betas, body = extractAndRemoveBetas(body)
	if len(betas) > 0 {
		mergeBetasToHeader(req, betas)
	}

	// 2. Disable thinking if tool_choice forces tool use
	// Anthropic API does not allow thinking when tool_choice is set to "any" or "tool"
	body = disableThinkingIfToolChoiceForced(body)

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

// mergeBetasToHeader merges extracted betas into Anthropic-Beta header.
// Existing header values are preserved, duplicates are avoided.
func mergeBetasToHeader(req *http.Request, betas []string) {
	if len(betas) == 0 {
		return
	}

	// Get existing header value
	existing := req.Header.Get("Anthropic-Beta")
	existingSet := make(map[string]bool)

	if existing != "" {
		for _, b := range strings.Split(existing, ",") {
			existingSet[strings.TrimSpace(b)] = true
		}
	}

	// Add new betas that don't already exist
	var newBetas []string
	for _, b := range betas {
		if !existingSet[b] {
			newBetas = append(newBetas, b)
			existingSet[b] = true
		}
	}

	// Merge all betas
	if len(newBetas) > 0 {
		var allBetas []string
		if existing != "" {
			allBetas = append(allBetas, existing)
		}
		allBetas = append(allBetas, newBetas...)
		req.Header.Set("Anthropic-Beta", strings.Join(allBetas, ","))
	}
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

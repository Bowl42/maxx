package antigravity

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BlockType represents the type of content block being processed
type BlockType int

const (
	BlockTypeNone BlockType = iota
	BlockTypeText
	BlockTypeThinking
	BlockTypeFunction
)

// ClaudeStreamingState maintains state for Gemini -> Claude SSE conversion
type ClaudeStreamingState struct {
	blockType        BlockType
	blockIndex       int
	messageStartSent bool
	messageStopSent  bool
	usedTool         bool

	// Signature management
	pendingSignature  *string
	trailingSignature *string

	// Token usage tracking
	inputTokens     int
	outputTokens    int
	cacheReadTokens int

	// Response metadata
	requestModel string // Original Claude model from request (for response)
	modelVersion string // Gemini model version from upstream (for debugging)
	responseID   string

	// Grounding (web search) captured during streaming, emitted at finish (like Antigravity-Manager)
	webSearchQuery  string
	groundingChunks []GeminiGroundingChunk
}

// NewClaudeStreamingState creates a new streaming state
func NewClaudeStreamingState() *ClaudeStreamingState {
	return &ClaudeStreamingState{
		blockType:  BlockTypeNone,
		blockIndex: 0,
	}
}

// NewClaudeStreamingStateWithSession creates a new streaming state with session ID and request model
func NewClaudeStreamingStateWithSession(_ string, requestModel string) *ClaudeStreamingState {
	return &ClaudeStreamingState{
		blockType:    BlockTypeNone,
		blockIndex:   0,
		requestModel: requestModel,
	}
}

// GetModelVersion returns the upstream model version captured during streaming
func (s *ClaudeStreamingState) GetModelVersion() string {
	return s.modelVersion
}

// GeminiPart represents a part in Gemini response
type GeminiPart struct {
	Text             string              `json:"text,omitempty"`
	Thought          bool                `json:"thought,omitempty"`
	ThoughtSignature string              `json:"thoughtSignature,omitempty"`
	FunctionCall     *GeminiFunctionCall `json:"functionCall,omitempty"`
	InlineData       *GeminiInlineData   `json:"inlineData,omitempty"`
}

// GeminiFunctionCall represents a function call in Gemini response
type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
	ID   string                 `json:"id,omitempty"`
}

// GeminiInlineData represents inline data (images) in Gemini response
type GeminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// GeminiUsageMetadata represents usage metadata in Gemini response
type GeminiUsageMetadata struct {
	PromptTokenCount        int `json:"promptTokenCount"`
	CandidatesTokenCount    int `json:"candidatesTokenCount"`
	CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"`
}

// GeminiStreamChunk represents a streaming chunk from Gemini
type GeminiStreamChunk struct {
	Candidates []struct {
		Content struct {
			Parts []GeminiPart `json:"parts"`
		} `json:"content"`
		FinishReason      string                   `json:"finishReason,omitempty"`
		GroundingMetadata *GeminiGroundingMetadata `json:"groundingMetadata,omitempty"`
	} `json:"candidates"`
	UsageMetadata *GeminiUsageMetadata `json:"usageMetadata,omitempty"`
	ModelVersion  string               `json:"modelVersion,omitempty"`
	ResponseID    string               `json:"responseId,omitempty"`
}

// GeminiGroundingMetadata represents grounding/web search metadata from Gemini
type GeminiGroundingMetadata struct {
	WebSearchQueries []string                `json:"webSearchQueries,omitempty"`
	GroundingChunks  []GeminiGroundingChunk  `json:"groundingChunks,omitempty"`
	SearchEntryPoint *GeminiSearchEntryPoint `json:"searchEntryPoint,omitempty"`
}

// GeminiGroundingChunk represents a grounding chunk (web search result)
type GeminiGroundingChunk struct {
	Web *GeminiGroundingWeb `json:"web,omitempty"`
}

// GeminiGroundingWeb represents web source in grounding chunk
type GeminiGroundingWeb struct {
	URI   string `json:"uri,omitempty"`
	Title string `json:"title,omitempty"`
}

// GeminiSearchEntryPoint represents search entry point
type GeminiSearchEntryPoint struct {
	RenderedContent string `json:"renderedContent,omitempty"`
}

// formatSSE formats an SSE event with proper double newline terminator
func formatSSE(eventType string, data interface{}) []byte {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	return []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(jsonBytes)))
}

// emit emits a single SSE event
func (s *ClaudeStreamingState) emit(eventType string, data map[string]interface{}) []byte {
	return formatSSE(eventType, data)
}

// emitDelta emits a content_block_delta event
func (s *ClaudeStreamingState) emitDelta(deltaType string, deltaContent map[string]interface{}) []byte {
	delta := map[string]interface{}{"type": deltaType}
	for k, v := range deltaContent {
		delta[k] = v
	}

	return s.emit("content_block_delta", map[string]interface{}{
		"type":  "content_block_delta",
		"index": s.blockIndex,
		"delta": delta,
	})
}

// emitMessageStart emits the message_start event
func (s *ClaudeStreamingState) emitMessageStart(chunk *GeminiStreamChunk) []byte {
	if s.messageStartSent {
		return nil
	}

	responseID := chunk.ResponseID
	if responseID == "" {
		responseID = "msg_unknown"
	}
	s.responseID = responseID

	if chunk.ModelVersion != "" {
		s.modelVersion = chunk.ModelVersion
	}

	message := map[string]interface{}{
		"id":            s.responseID,
		"type":          "message",
		"role":          "assistant",
		"content":       []interface{}{},
		"model":         s.modelVersion, // Use upstream model version (like Antigravity-Manager)
		"stop_reason":   nil,
		"stop_sequence": nil,
	}

	// Usage is only present when upstream provides usageMetadata (like Antigravity-Manager)
	if chunk.UsageMetadata != nil {
		cachedTokens := chunk.UsageMetadata.CachedContentTokenCount
		inputTokens := chunk.UsageMetadata.PromptTokenCount - cachedTokens
		if inputTokens < 0 {
			inputTokens = 0
		}

		usage := map[string]interface{}{
			"input_tokens":                inputTokens,
			"output_tokens":               chunk.UsageMetadata.CandidatesTokenCount,
			"cache_creation_input_tokens": 0, // Gemini doesn't provide this, set to 0
		}
		if cachedTokens > 0 {
			usage["cache_read_input_tokens"] = cachedTokens
		}
		message["usage"] = usage
	}

	result := s.emit("message_start", map[string]interface{}{
		"type":    "message_start",
		"message": message,
	})

	s.messageStartSent = true
	return result
}

// startBlock starts a new content block
func (s *ClaudeStreamingState) startBlock(blockType BlockType, contentBlock map[string]interface{}) [][]byte {
	var chunks [][]byte

	// End previous block if any
	if s.blockType != BlockTypeNone {
		chunks = append(chunks, s.endBlock()...)
	}

	// Start new block
	chunks = append(chunks, s.emit("content_block_start", map[string]interface{}{
		"type":          "content_block_start",
		"index":         s.blockIndex,
		"content_block": contentBlock,
	}))

	s.blockType = blockType
	return chunks
}

// endBlock ends the current content block
func (s *ClaudeStreamingState) endBlock() [][]byte {
	if s.blockType == BlockTypeNone {
		return nil
	}

	var chunks [][]byte

	// Emit pending signature for thinking blocks
	if s.blockType == BlockTypeThinking && s.pendingSignature != nil {
		chunks = append(chunks, s.emitDelta("signature_delta", map[string]interface{}{
			"signature": *s.pendingSignature,
		}))
		s.pendingSignature = nil
	}

	chunks = append(chunks, s.emit("content_block_stop", map[string]interface{}{
		"type":  "content_block_stop",
		"index": s.blockIndex,
	}))

	s.blockIndex++
	s.blockType = BlockTypeNone

	return chunks
}

// emitFinish emits the finish events (message_delta and message_stop)
func (s *ClaudeStreamingState) emitFinish(finishReason string, usage *GeminiUsageMetadata) [][]byte {
	var chunks [][]byte

	// End current block
	chunks = append(chunks, s.endBlock()...)

	// Handle trailing signature
	if s.trailingSignature != nil {
		// Create empty thinking block for trailing signature
		chunks = append(chunks, s.emit("content_block_start", map[string]interface{}{
			"type":  "content_block_start",
			"index": s.blockIndex,
			"content_block": map[string]interface{}{
				"type":     "thinking",
				"thinking": "",
			},
		}))
		chunks = append(chunks, s.emitDelta("thinking_delta", map[string]interface{}{"thinking": ""}))
		chunks = append(chunks, s.emitDelta("signature_delta", map[string]interface{}{"signature": *s.trailingSignature}))
		chunks = append(chunks, s.emit("content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": s.blockIndex,
		}))
		s.blockIndex++
		s.trailingSignature = nil
	}

	// Grounding (web search) -> emit as a separate Markdown text block at finish (like Antigravity-Manager)
	if groundingText := s.buildGroundingMarkdown(); groundingText != "" {
		chunks = append(chunks, s.emit("content_block_start", map[string]interface{}{
			"type":  "content_block_start",
			"index": s.blockIndex,
			"content_block": map[string]interface{}{
				"type": "text",
				"text": "",
			},
		}))
		chunks = append(chunks, s.emitDelta("text_delta", map[string]interface{}{"text": groundingText}))
		chunks = append(chunks, s.emit("content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": s.blockIndex,
		}))
		s.blockIndex++

		// Clear grounding so we don't emit twice
		s.webSearchQuery = ""
		s.groundingChunks = nil
	}

	// Determine stop reason
	stopReason := "end_turn"
	if s.usedTool {
		stopReason = "tool_use"
	} else if finishReason == "MAX_TOKENS" {
		stopReason = "max_tokens"
	}

	// Build usage with all fields (like Antigravity-Manager's to_claude_usage)
	usageMap := map[string]interface{}{
		"input_tokens":  s.inputTokens,
		"output_tokens": s.outputTokens,
	}
	if usage != nil {
		cachedTokens := usage.CachedContentTokenCount
		inputTokens := usage.PromptTokenCount - cachedTokens
		if inputTokens < 0 {
			inputTokens = 0
		}
		usageMap["input_tokens"] = inputTokens
		usageMap["output_tokens"] = usage.CandidatesTokenCount
		if cachedTokens > 0 {
			usageMap["cache_read_input_tokens"] = cachedTokens
		}
		// cache_creation_input_tokens: Gemini doesn't provide this, set to 0 (like Antigravity-Manager)
		usageMap["cache_creation_input_tokens"] = 0
	}

	chunks = append(chunks, s.emit("message_delta", map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason":   stopReason,
			"stop_sequence": nil,
		},
		"usage": usageMap,
	}))

	if !s.messageStopSent {
		chunks = append(chunks, []byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
		s.messageStopSent = true
	}

	return chunks
}

// storeSignature stores a pending signature and caches it for future requests
func (s *ClaudeStreamingState) storeSignature(signature string) {
	if signature != "" {
		s.pendingSignature = &signature

		// Cache thinking family for cross-model compatibility (like Antigravity-Manager)
		if s.modelVersion != "" {
			GlobalSignatureCache().CacheThinkingFamily(signature, s.modelVersion)
		}

		// Best-effort global fallback store
		StoreThoughtSignature(signature)
	}
}

// setTrailingSignature sets the trailing signature
func (s *ClaudeStreamingState) setTrailingSignature(signature string) {
	if signature != "" {
		s.trailingSignature = &signature
	}
}

// hasTrailingSignature checks if there's a trailing signature
func (s *ClaudeStreamingState) hasTrailingSignature() bool {
	return s.trailingSignature != nil
}

// markToolUsed marks that a tool was used
func (s *ClaudeStreamingState) markToolUsed() {
	s.usedTool = true
}

// processThinking processes a thinking part
func (s *ClaudeStreamingState) processThinking(text, signature string) [][]byte {
	var chunks [][]byte

	// Handle previous trailing signature
	if s.hasTrailingSignature() {
		chunks = append(chunks, s.endBlock()...)
		chunks = append(chunks, s.emitEmptyThinkingWithSignature(*s.trailingSignature)...)
		s.trailingSignature = nil
	}

	// Start thinking block if not already in one
	if s.blockType != BlockTypeThinking {
		chunks = append(chunks, s.startBlock(BlockTypeThinking, map[string]interface{}{
			"type":     "thinking",
			"thinking": "",
		})...)
	}

	// Emit thinking content
	if text != "" {
		chunks = append(chunks, s.emitDelta("thinking_delta", map[string]interface{}{
			"thinking": text,
		}))
	}

	// Store signature for later (also caches it)
	if signature != "" {
		s.storeSignature(signature)
	}

	return chunks
}

// processText processes a text part
func (s *ClaudeStreamingState) processText(text, signature string) [][]byte {
	var chunks [][]byte

	// Empty text with signature -> trailing signature
	if text == "" {
		if signature != "" {
			s.setTrailingSignature(signature)
		}
		return chunks
	}

	// Handle previous trailing signature
	if s.hasTrailingSignature() {
		chunks = append(chunks, s.endBlock()...)
		chunks = append(chunks, s.emitEmptyThinkingWithSignature(*s.trailingSignature)...)
		s.trailingSignature = nil
	}

	// Non-empty text with signature -> emit text, then empty thinking with signature
	if signature != "" {
		// Start and emit text
		chunks = append(chunks, s.startBlock(BlockTypeText, map[string]interface{}{
			"type": "text",
			"text": "",
		})...)
		chunks = append(chunks, s.emitDelta("text_delta", map[string]interface{}{"text": text}))
		chunks = append(chunks, s.endBlock()...)

		// Emit empty thinking block with signature
		chunks = append(chunks, s.emitEmptyThinkingWithSignature(signature)...)
		return chunks
	}

	// Regular text (no signature)
	if s.blockType != BlockTypeText {
		chunks = append(chunks, s.startBlock(BlockTypeText, map[string]interface{}{
			"type": "text",
			"text": "",
		})...)
	}

	chunks = append(chunks, s.emitDelta("text_delta", map[string]interface{}{"text": text}))

	return chunks
}

// emitEmptyThinkingWithSignature emits an empty thinking block with signature
func (s *ClaudeStreamingState) emitEmptyThinkingWithSignature(signature string) [][]byte {
	var chunks [][]byte

	chunks = append(chunks, s.emit("content_block_start", map[string]interface{}{
		"type":  "content_block_start",
		"index": s.blockIndex,
		"content_block": map[string]interface{}{
			"type":     "thinking",
			"thinking": "",
		},
	}))
	chunks = append(chunks, s.emitDelta("thinking_delta", map[string]interface{}{"thinking": ""}))
	chunks = append(chunks, s.emitDelta("signature_delta", map[string]interface{}{"signature": signature}))
	chunks = append(chunks, s.emit("content_block_stop", map[string]interface{}{
		"type":  "content_block_stop",
		"index": s.blockIndex,
	}))
	s.blockIndex++

	return chunks
}

// processFunctionCall processes a function call part
func (s *ClaudeStreamingState) processFunctionCall(fc *GeminiFunctionCall, signature string) [][]byte {
	var chunks [][]byte

	// Handle trailing signature first
	if s.hasTrailingSignature() {
		chunks = append(chunks, s.endBlock()...)
		chunks = append(chunks, s.emitEmptyThinkingWithSignature(*s.trailingSignature)...)
		s.trailingSignature = nil
	}

	s.markToolUsed()

	// Generate tool ID
	toolID := fc.ID
	if toolID == "" {
		toolID = fmt.Sprintf("%s-%d", fc.Name, generateRandomID())
	}

	// [FIX] Cache tool_id -> signature mapping (like Antigravity-Manager)
	// This allows future requests to recover the signature for this tool call
	if signature != "" && len(signature) >= MinSignatureLength {
		GlobalSignatureCache().CacheToolSignature(toolID, signature)
	}

	// Build tool_use content block
	toolUse := map[string]interface{}{
		"type":  "tool_use",
		"id":    toolID,
		"name":  fc.Name,
		"input": map[string]interface{}{}, // Empty, args sent via delta
	}

	if signature != "" {
		toolUse["signature"] = signature
	}

	// Start tool_use block
	chunks = append(chunks, s.startBlock(BlockTypeFunction, toolUse)...)

	// Emit input_json_delta with remapped arguments
	if fc.Args != nil {
		args := fc.Args
		remapFunctionCallArgs(fc.Name, args)
		argsJSON, _ := json.Marshal(args)
		chunks = append(chunks, s.emitDelta("input_json_delta", map[string]interface{}{
			"partial_json": string(argsJSON),
		}))
	}

	// End tool block
	chunks = append(chunks, s.endBlock()...)

	return chunks
}

// EmitForceStop ensures all termination events are sent
// Called when stream ends (EOF or [DONE])
func (s *ClaudeStreamingState) EmitForceStop() []byte {
	if s.messageStopSent {
		return nil
	}

	var output []byte
	finishChunks := s.emitFinish("", nil)
	for _, c := range finishChunks {
		output = append(output, c...)
	}

	// If emitFinish didn't send message_stop, send it now
	if !s.messageStopSent {
		output = append(output, []byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")...)
		s.messageStopSent = true
	}

	return output
}

// ProcessGeminiSSELine processes a single Gemini SSE line and returns Claude SSE events
func (s *ClaudeStreamingState) ProcessGeminiSSELine(line string) []byte {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	if !strings.HasPrefix(line, "data: ") {
		return nil
	}

	dataStr := strings.TrimPrefix(line, "data: ")
	dataStr = strings.TrimSpace(dataStr)
	if dataStr == "" {
		return nil
	}

	// Handle [DONE] signal - emit force stop events
	if dataStr == "[DONE]" {
		return s.EmitForceStop()
	}

	// Parse Gemini chunk
	var chunk GeminiStreamChunk
	if err := json.Unmarshal([]byte(dataStr), &chunk); err != nil {
		// [SSE Error Recovery] Handle parse errors gracefully (like Antigravity-Manager)
		// Instead of silently dropping, try to extract partial data or log
		return s.handleParseError(dataStr, err)
	}

	var output []byte

	// Send message_start on first chunk
	if !s.messageStartSent {
		if data := s.emitMessageStart(&chunk); data != nil {
			output = append(output, data...)
		}
	}

	// Update usage metadata (input_tokens should exclude cached tokens)
	if chunk.UsageMetadata != nil {
		cachedTokens := chunk.UsageMetadata.CachedContentTokenCount
		inputTokens := chunk.UsageMetadata.PromptTokenCount - cachedTokens
		if inputTokens < 0 {
			inputTokens = 0
		}
		s.inputTokens = inputTokens
		s.outputTokens = chunk.UsageMetadata.CandidatesTokenCount
		s.cacheReadTokens = cachedTokens
	}

	// Process candidates
	if len(chunk.Candidates) > 0 {
		candidate := chunk.Candidates[0]

		// Process each part
		for _, part := range candidate.Content.Parts {
			chunks := s.processPart(&part)
			for _, c := range chunks {
				output = append(output, c...)
			}
		}

		// Process grounding metadata (web search results) - like Antigravity-Manager
		if candidate.GroundingMetadata != nil {
			s.captureGrounding(candidate.GroundingMetadata)
		}

		// Handle finish
		if candidate.FinishReason != "" {
			finishChunks := s.emitFinish(candidate.FinishReason, chunk.UsageMetadata)
			for _, c := range finishChunks {
				output = append(output, c...)
			}
		}
	}

	return output
}

// processPart processes a single Gemini part
func (s *ClaudeStreamingState) processPart(part *GeminiPart) [][]byte {
	signature := part.ThoughtSignature

	// 1. Handle function call
	if part.FunctionCall != nil {
		return s.processFunctionCall(part.FunctionCall, signature)
	}

	// 2. Handle text/thinking
	if part.Text != "" || signature != "" {
		if part.Thought {
			return s.processThinking(part.Text, signature)
		}
		return s.processText(part.Text, signature)
	}

	// 3. Handle inline data (images)
	if part.InlineData != nil && part.InlineData.Data != "" {
		markdownImg := fmt.Sprintf("![image](data:%s;base64,%s)", part.InlineData.MimeType, part.InlineData.Data)
		return s.processText(markdownImg, "")
	}

	return nil
}

// captureGrounding stores grounding metadata during streaming, to be emitted at finish.
func (s *ClaudeStreamingState) captureGrounding(grounding *GeminiGroundingMetadata) {
	if grounding == nil {
		return
	}
	if len(grounding.WebSearchQueries) > 0 {
		s.webSearchQuery = grounding.WebSearchQueries[0]
	}
	if len(grounding.GroundingChunks) > 0 {
		s.groundingChunks = grounding.GroundingChunks
	}
}

// buildGroundingMarkdown builds grounding(web search) markdown text (same format as Antigravity-Manager).
func (s *ClaudeStreamingState) buildGroundingMarkdown() string {
	if s.webSearchQuery == "" && len(s.groundingChunks) == 0 {
		return ""
	}

	var groundingText strings.Builder

	// 1. Search query
	if strings.TrimSpace(s.webSearchQuery) != "" {
		groundingText.WriteString("\n\n---\n**🔍 已为您搜索：** ")
		groundingText.WriteString(s.webSearchQuery)
	}

	// 2. Source links
	if len(s.groundingChunks) > 0 {
		links := make([]string, 0, len(s.groundingChunks))
		for i, chunk := range s.groundingChunks {
			if chunk.Web == nil {
				continue
			}
			title := chunk.Web.Title
			if title == "" {
				title = "网页来源"
			}
			uri := chunk.Web.URI
			if uri == "" {
				uri = "#"
			}
			links = append(links, fmt.Sprintf("[%d] [%s](%s)", i+1, title, uri))
		}

		if len(links) > 0 {
			groundingText.WriteString("\n\n**🌐 来源引文：**\n")
			groundingText.WriteString(strings.Join(links, "\n"))
		}
	}

	return groundingText.String()
}

// remapFunctionCallArgs remaps Gemini function call arguments to Claude Code expected format
func remapFunctionCallArgs(toolName string, args map[string]interface{}) {
	if args == nil {
		return
	}

	toolNameLower := strings.ToLower(toolName)

	switch toolNameLower {
	case "grep":
		// Gemini uses "query", Claude Code expects "pattern"
		if query, ok := args["query"]; ok {
			if _, hasPattern := args["pattern"]; !hasPattern {
				args["pattern"] = query
				delete(args, "query")
			}
		}
		// Claude Code uses "path" (string), NOT "paths" (array)
		if _, hasPath := args["path"]; !hasPath {
			if paths, ok := args["paths"]; ok {
				pathStr := extractFirstPath(paths)
				args["path"] = pathStr
				delete(args, "paths")
			} else {
				args["path"] = "."
			}
		}

	case "glob":
		// Gemini uses "query", Claude Code expects "pattern"
		if query, ok := args["query"]; ok {
			if _, hasPattern := args["pattern"]; !hasPattern {
				args["pattern"] = query
				delete(args, "query")
			}
		}
		// Claude Code uses "path" (string), NOT "paths" (array)
		if _, hasPath := args["path"]; !hasPath {
			if paths, ok := args["paths"]; ok {
				pathStr := extractFirstPath(paths)
				args["path"] = pathStr
				delete(args, "paths")
			} else {
				args["path"] = "."
			}
		}

	case "read":
		// Gemini might use "path" vs "file_path"
		if path, ok := args["path"]; ok {
			if _, hasFilePath := args["file_path"]; !hasFilePath {
				args["file_path"] = path
				delete(args, "path")
			}
		}

	case "ls":
		// LS tool: ensure "path" parameter exists
		if _, hasPath := args["path"]; !hasPath {
			args["path"] = "."
		}
	}
}

// extractFirstPath extracts the first path from various input formats
func extractFirstPath(paths interface{}) string {
	switch v := paths.(type) {
	case []interface{}:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
		return "."
	case string:
		return v
	default:
		return "."
	}
}

// generateRandomID generates a simple random ID using time
func generateRandomID() int64 {
	return time.Now().UnixNano()
}

// handleParseError handles SSE parse errors gracefully (like Antigravity-Manager's handle_parse_error)
// Attempts to recover partial data or emits a warning text block
func (s *ClaudeStreamingState) handleParseError(dataStr string, err error) []byte {
	// Try to extract error message from the data if it's an error response
	if strings.Contains(dataStr, "error") {
		// Attempt to parse as error response
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(dataStr), &errorResp) == nil && errorResp.Error.Message != "" {
			// Emit error as text content
			errorText := fmt.Sprintf("\n\n[API Error: %s (code: %d, status: %s)]\n",
				errorResp.Error.Message, errorResp.Error.Code, errorResp.Error.Status)

			var output []byte
			// Ensure message_start is sent
			if !s.messageStartSent {
				startData := s.emitMessageStart(&GeminiStreamChunk{})
				if startData != nil {
					output = append(output, startData...)
				}
			}

			// Emit as text block
			textChunks := s.processText(errorText, "")
			for _, c := range textChunks {
				output = append(output, c...)
			}
			return output
		}
	}

	// For other parse errors, try partial text extraction
	if strings.Contains(dataStr, "\"text\"") {
		// Try to extract text field directly using regex-like approach
		textStart := strings.Index(dataStr, "\"text\":\"")
		if textStart >= 0 {
			textStart += 8
			textEnd := strings.Index(dataStr[textStart:], "\"")
			if textEnd > 0 {
				partialText := dataStr[textStart : textStart+textEnd]
				// Unescape basic JSON escapes
				partialText = strings.ReplaceAll(partialText, "\\n", "\n")
				partialText = strings.ReplaceAll(partialText, "\\t", "\t")
				partialText = strings.ReplaceAll(partialText, "\\\"", "\"")

				var output []byte
				if !s.messageStartSent {
					startData := s.emitMessageStart(&GeminiStreamChunk{})
					if startData != nil {
						output = append(output, startData...)
					}
				}
				textChunks := s.processText(partialText, "")
				for _, c := range textChunks {
					output = append(output, c...)
				}
				return output
			}
		}
	}

	// Cannot recover - return nil (drop the malformed chunk)
	return nil
}

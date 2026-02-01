package converter

// Claude API types

type ClaudeRequest struct {
	Model         string                 `json:"model"`
	Messages      []ClaudeMessage        `json:"messages"`
	System        interface{}            `json:"system,omitempty"` // string or []SystemBlock
	MaxTokens     int                    `json:"max_tokens,omitempty"`
	Temperature   *float64               `json:"temperature,omitempty"`
	TopP          *float64               `json:"top_p,omitempty"`
	TopK          *int                   `json:"top_k,omitempty"`
	StopSequences []string               `json:"stop_sequences,omitempty"`
	Stream        bool                   `json:"stream,omitempty"`
	Metadata      *ClaudeMetadata        `json:"metadata,omitempty"`
	Tools         []ClaudeTool           `json:"tools,omitempty"`
	ToolChoice    interface{}            `json:"tool_choice,omitempty"`
	Thinking      map[string]interface{} `json:"thinking,omitempty"` // {"type": "enabled", "budget_tokens": N}
	OutputConfig  *ClaudeOutputConfig    `json:"output_config,omitempty"`
}

// ClaudeMetadata represents request metadata (like Antigravity-Manager)
type ClaudeMetadata struct {
	UserID string `json:"user_id,omitempty"` // Used as sessionId for v1internal
}

// ClaudeOutputConfig represents output configuration for effort level (Claude API v2.0.67+)
type ClaudeOutputConfig struct {
	Effort string `json:"effort,omitempty"` // "high", "medium", "low"
}

type ClaudeMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []ContentBlock
}

type ClaudeContentBlock struct {
	Type      string      `json:"type"`
	Text      string      `json:"text,omitempty"`
	ID        string      `json:"id,omitempty"`
	Name      string      `json:"name,omitempty"`
	Input     interface{} `json:"input,omitempty"`
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"` // string or []ContentBlock for tool_result
	IsError   *bool       `json:"is_error,omitempty"`
	// Thinking block fields
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
	// RedactedThinking block field
	Data string `json:"data,omitempty"`
	// Cache control (will be cleaned before sending to upstream)
	CacheControl interface{} `json:"cache_control,omitempty"`
	// Image source
	Source *ClaudeImageSource `json:"source,omitempty"`
}

// ClaudeImageSource represents image source in Claude API
type ClaudeImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // e.g. "image/png"
	Data      string `json:"data"`       // base64 data
}

type ClaudeTool struct {
	Type        string      `json:"type,omitempty"` // For server tools like "web_search_20250305"
	Name        string      `json:"name,omitempty"` // Tool name
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"input_schema,omitempty"` // Required for client tools, absent for server tools
}

// IsWebSearch checks if this is the web_search server tool
func (t *ClaudeTool) IsWebSearch() bool {
	// Check by type (preferred for server tools)
	if t.Type != "" {
		if len(t.Type) >= 10 && t.Type[:10] == "web_search" {
			return true
		}
	}
	// Check by name (fallback)
	if t.Name == "web_search" || t.Name == "google_search" || t.Name == "google_search_retrieval" {
		return true
	}
	return false
}

type ClaudeResponse struct {
	ID           string               `json:"id"`
	Type         string               `json:"type"`
	Role         string               `json:"role"`
	Content      []ClaudeContentBlock `json:"content"`
	Model        string               `json:"model"`
	StopReason   string               `json:"stop_reason"`
	StopSequence string               `json:"stop_sequence,omitempty"`
	Usage        ClaudeUsage          `json:"usage"`
}

type ClaudeUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
}

// Claude streaming events
type ClaudeStreamEvent struct {
	Type         string              `json:"type"`
	Message      *ClaudeResponse     `json:"message,omitempty"`
	Index        int                 `json:"index,omitempty"`
	ContentBlock *ClaudeContentBlock `json:"content_block,omitempty"`
	Delta        *ClaudeStreamDelta  `json:"delta,omitempty"`
	Usage        *ClaudeUsage        `json:"usage,omitempty"`
}

type ClaudeStreamDelta struct {
	Type         string `json:"type,omitempty"`
	Text         string `json:"text,omitempty"`
	Thinking     string `json:"thinking,omitempty"`
	PartialJSON  string `json:"partial_json,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}

package kiro

import "time"

// ValueType represents AWS Event Stream header value types.
type ValueType byte

const (
	ValueTypeBoolTrue  ValueType = 0
	ValueTypeBoolFalse ValueType = 1
	ValueTypeByte      ValueType = 2
	ValueTypeShort     ValueType = 3
	ValueTypeInteger   ValueType = 4
	ValueTypeLong      ValueType = 5
	ValueTypeByteArray ValueType = 6
	ValueTypeString    ValueType = 7
	ValueTypeTimestamp ValueType = 8
	ValueTypeUUID      ValueType = 9
)

// HeaderValue stores a parsed header value.
type HeaderValue struct {
	Type  ValueType
	Value any
}

// EventStreamMessage represents a decoded EventStream message.
type EventStreamMessage struct {
	Headers     map[string]HeaderValue
	Payload     []byte
	MessageType string
	EventType   string
	ContentType string
}

// GetMessageType returns the message type from headers.
func (esm *EventStreamMessage) GetMessageType() string {
	if header, ok := esm.Headers[":message-type"]; ok {
		if msgType, ok := header.Value.(string); ok {
			return msgType
		}
	}
	return "event"
}

// GetEventType returns the event type from headers.
func (esm *EventStreamMessage) GetEventType() string {
	if header, ok := esm.Headers[":event-type"]; ok {
		if eventType, ok := header.Value.(string); ok {
			return eventType
		}
	}
	return ""
}

// GetContentType returns the content type from headers.
func (esm *EventStreamMessage) GetContentType() string {
	if header, ok := esm.Headers[":content-type"]; ok {
		if contentType, ok := header.Value.(string); ok {
			return contentType
		}
	}
	return "application/json"
}

// MessageTypes defines the core EventStream message types.
var MessageTypes = struct {
	Event     string
	Error     string
	Exception string
}{
	Event:     "event",
	Error:     "error",
	Exception: "exception",
}

// EventTypes defines known event types from CodeWhisperer.
var EventTypes = struct {
	Completion       string
	CompletionChunk  string
	ToolCallRequest  string
	ToolCallResult   string
	ToolCallError    string
	SessionStart     string
	SessionEnd       string
	AssistantEvent   string
	ToolUseEvent     string
}{
	Completion:      "completion",
	CompletionChunk: "completion_chunk",
	ToolCallRequest: "tool_call_request",
	ToolCallResult:  "tool_call_result",
	ToolCallError:   "tool_call_error",
	SessionStart:    "session_start",
	SessionEnd:      "session_end",
	AssistantEvent:  "assistantResponseEvent",
	ToolUseEvent:    "toolUseEvent",
}

// ToolExecution tracks tool lifecycle state for streaming reconstruction.
type ToolExecution struct {
	ID         string
	Name       string
	StartTime  time.Time
	EndTime    *time.Time
	Status     ToolExecutionStatus
	Arguments  map[string]any
	Result     any
	Error      string
	BlockIndex int
}

// ToolExecutionStatus tracks execution state.
type ToolExecutionStatus int

const (
	ToolStatusPending ToolExecutionStatus = iota
	ToolStatusRunning
	ToolStatusCompleted
	ToolStatusError
)

func (s ToolExecutionStatus) String() string {
	switch s {
	case ToolStatusPending:
		return "pending"
	case ToolStatusRunning:
		return "running"
	case ToolStatusCompleted:
		return "completed"
	case ToolStatusError:
		return "error"
	default:
		return "unknown"
	}
}

// ToolCallRequest represents a tool call request event payload.
type ToolCallRequest struct {
	ToolCalls []ToolCall `json:"tool_calls"`
}

// ToolCall represents a single tool invocation.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction represents the function for a tool call.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCallResult represents a tool call result payload.
type ToolCallResult struct {
	ToolCallID    string `json:"tool_call_id"`
	Result        any    `json:"result"`
	ExecutionTime int64  `json:"execution_time,omitempty"`
}

// ToolCallError represents a tool call error payload.
type ToolCallError struct {
	ToolCallID string `json:"tool_call_id"`
	Error      string `json:"error"`
}

// SessionInfo represents session metadata.
type SessionInfo struct {
	SessionID string     `json:"session_id"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// ParseError represents a parser error.
type ParseError struct {
	Message string
	Cause   error
}

func (e *ParseError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return e.Message + ": " + e.Cause.Error()
}

func NewParseError(message string, cause error) *ParseError {
	return &ParseError{Message: message, Cause: cause}
}

const (
	EventStreamMinMessageSize = 16
	EventStreamMaxMessageSize = 16 * 1024 * 1024
)

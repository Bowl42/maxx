package kiro

import "strings"

// EventHandler processes a single EventStreamMessage.
type EventHandler interface {
	Handle(message *EventStreamMessage) ([]SSEEvent, error)
}

// CompliantMessageProcessor converts EventStream messages to Claude SSE events.
type CompliantMessageProcessor struct {
	sessionManager     *SessionManager
	toolManager        *ToolLifecycleManager
	eventHandlers      map[string]EventHandler
	completionBuffer   []string
	toolDataAggregator *StreamingJSONAggregator
}

// NewCompliantMessageProcessor creates a processor.
func NewCompliantMessageProcessor() *CompliantMessageProcessor {
	processor := &CompliantMessageProcessor{
		sessionManager:   NewSessionManager(),
		toolManager:      NewToolLifecycleManager(),
		eventHandlers:    make(map[string]EventHandler),
		completionBuffer: make([]string, 0, 16),
	}

	processor.toolDataAggregator = NewStreamingJSONAggregatorWithCallback(
		func(toolUseID string, fullParams string) {
			processor.toolManager.UpdateToolArgumentsFromJSON(toolUseID, fullParams)
		},
	)

	processor.registerEventHandlers()
	return processor
}

// Reset clears processor state.
func (cmp *CompliantMessageProcessor) Reset() {
	cmp.sessionManager.Reset()
	cmp.toolManager.Reset()
	cmp.completionBuffer = cmp.completionBuffer[:0]
}

func (cmp *CompliantMessageProcessor) registerEventHandlers() {
	cmp.eventHandlers[EventTypes.Completion] = &CompletionEventHandler{processor: cmp}
	cmp.eventHandlers[EventTypes.CompletionChunk] = &CompletionChunkEventHandler{processor: cmp}
	cmp.eventHandlers[EventTypes.ToolCallRequest] = &ToolCallRequestHandler{toolManager: cmp.toolManager}
	cmp.eventHandlers[EventTypes.ToolCallError] = &ToolCallErrorHandler{toolManager: cmp.toolManager}
	cmp.eventHandlers[EventTypes.SessionStart] = &SessionStartHandler{sessionManager: cmp.sessionManager}
	cmp.eventHandlers[EventTypes.SessionEnd] = &SessionEndHandler{sessionManager: cmp.sessionManager}
	cmp.eventHandlers[EventTypes.AssistantEvent] = &StandardAssistantResponseEventHandler{processor: cmp}
	cmp.eventHandlers[EventTypes.ToolUseEvent] = &LegacyToolUseEventHandler{
		toolManager: cmp.toolManager,
		aggregator:  cmp.toolDataAggregator,
	}
}

// ProcessMessage converts a message to SSE events.
func (cmp *CompliantMessageProcessor) ProcessMessage(message *EventStreamMessage) ([]SSEEvent, error) {
	messageType := message.GetMessageType()
	eventType := message.GetEventType()

	switch messageType {
	case MessageTypes.Event:
		return cmp.processEventMessage(message, eventType)
	case MessageTypes.Error:
		return cmp.processErrorMessage(message)
	case MessageTypes.Exception:
		return cmp.processExceptionMessage(message)
	default:
		return []SSEEvent{}, nil
	}
}

func (cmp *CompliantMessageProcessor) processEventMessage(message *EventStreamMessage, eventType string) ([]SSEEvent, error) {
	if handler, exists := cmp.eventHandlers[eventType]; exists {
		return handler.Handle(message)
	}
	return []SSEEvent{}, nil
}

func (cmp *CompliantMessageProcessor) processErrorMessage(message *EventStreamMessage) ([]SSEEvent, error) {
	errorData := map[string]any{}
	if len(message.Payload) > 0 {
		if err := jsonUnmarshal(message.Payload, &errorData); err != nil {
			errorData = map[string]any{"message": string(message.Payload)}
		}
	}

	errorCode := ""
	errorMessage := ""
	if code, ok := errorData["__type"].(string); ok {
		errorCode = code
	}
	if msg, ok := errorData["message"].(string); ok {
		errorMessage = msg
	}

	return []SSEEvent{
		{
			Event: "error",
			Data: map[string]any{
				"type":          "error",
				"error_code":    errorCode,
				"error_message": errorMessage,
				"raw_data":      errorData,
			},
		},
	}, nil
}

func (cmp *CompliantMessageProcessor) processExceptionMessage(message *EventStreamMessage) ([]SSEEvent, error) {
	exceptionData := map[string]any{}
	if len(message.Payload) > 0 {
		if err := jsonUnmarshal(message.Payload, &exceptionData); err != nil {
			exceptionData = map[string]any{"message": string(message.Payload)}
		}
	}

	exceptionType := ""
	exceptionMessage := ""
	if eType, ok := exceptionData["__type"].(string); ok {
		exceptionType = eType
	}
	if msg, ok := exceptionData["message"].(string); ok {
		exceptionMessage = msg
	}

	return []SSEEvent{
		{
			Event: "exception",
			Data: map[string]any{
				"type":              "exception",
				"exception_type":    exceptionType,
				"exception_message": exceptionMessage,
				"raw_data":          exceptionData,
			},
		},
	}, nil
}

// GetSessionManager returns the session manager.
func (cmp *CompliantMessageProcessor) GetSessionManager() *SessionManager {
	return cmp.sessionManager
}

// GetCompletionBuffer returns the aggregated completion text.
func (cmp *CompliantMessageProcessor) GetCompletionBuffer() string {
	if len(cmp.completionBuffer) == 0 {
		return ""
	}
	return strings.Join(cmp.completionBuffer, "")
}

// GetToolManager returns the tool lifecycle manager.
func (cmp *CompliantMessageProcessor) GetToolManager() *ToolLifecycleManager {
	return cmp.toolManager
}

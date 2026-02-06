package context

type contextKey string

const (
	CtxKeyClientType         contextKey = "client_type"
	CtxKeyOriginalClientType contextKey = "original_client_type"
	CtxKeySessionID          contextKey = "session_id"
	CtxKeyProjectID          contextKey = "project_id"
	CtxKeyRequestModel       contextKey = "request_model"
	CtxKeyMappedModel        contextKey = "mapped_model"
	CtxKeyResponseModel      contextKey = "response_model"
	CtxKeyProxyRequest       contextKey = "proxy_request"
	CtxKeyRequestBody        contextKey = "request_body"
	CtxKeyUpstreamAttempt    contextKey = "upstream_attempt"
	CtxKeyRequestHeaders     contextKey = "request_headers"
	CtxKeyRequestURI         contextKey = "request_uri"
	CtxKeyBroadcaster        contextKey = "broadcaster"
	CtxKeyIsStream           contextKey = "is_stream"
	CtxKeyAPITokenID         contextKey = "api_token_id"
	CtxKeyEventChan          contextKey = "event_chan"
)

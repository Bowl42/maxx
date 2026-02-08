package converter

import (
	"encoding/json"
	"github.com/awsl-project/maxx/internal/domain"
	"strings"
	"testing"
)

func TestGlobalRegistryAndMustMarshal(t *testing.T) {
	if GetGlobalRegistry() == nil {
		t.Fatalf("global registry nil")
	}
	b := mustMarshal(map[string]interface{}{"k": "v"})
	if !strings.Contains(string(b), "k") {
		t.Fatalf("mustMarshal missing")
	}
}

func TestRemapFunctionCallArgsAndCollectReasoningText(t *testing.T) {
	args := map[string]interface{}{"query": "q", "paths": []interface{}{"a"}}
	remapFunctionCallArgs("grep", args)
	if args["pattern"] != "q" || args["path"] != "a" {
		t.Fatalf("grep remap failed: %#v", args)
	}
	args = map[string]interface{}{"query": "q"}
	remapFunctionCallArgs("glob", args)
	if args["pattern"] != "q" {
		t.Fatalf("glob remap failed: %#v", args)
	}
	args = map[string]interface{}{"path": "x"}
	remapFunctionCallArgs("read", args)
	if args["file_path"] != "x" {
		t.Fatalf("read remap failed: %#v", args)
	}
	args = map[string]interface{}{}
	remapFunctionCallArgs("ls", args)
	if args["path"] != "." {
		t.Fatalf("ls remap failed: %#v", args)
	}

	raw := []interface{}{map[string]interface{}{"text": "a"}, map[string]interface{}{"text": "b"}}
	if collectReasoningText(raw) != "ab" {
		t.Fatalf("collectReasoningText array")
	}
}

func TestSplitFunctionNameFallback(t *testing.T) {
	name, callID := splitFunctionName("tool")
	if name != "tool" || callID != "" {
		t.Fatalf("expected fallback split")
	}
	name, callID = splitFunctionName("tool_call_123")
	if name != "tool" || callID != "call_123" {
		t.Fatalf("expected call suffix")
	}
}

func TestHelperBranches(t *testing.T) {
	if parseInlineImage("data:image/png;base64,%%%") != nil {
		t.Fatalf("expected invalid base64")
	}
	badFile := map[string]interface{}{"file": map[string]interface{}{"filename": "a.unknown", "file_data": "Zg=="}}
	if parseFilePart(badFile) != nil {
		t.Fatalf("expected unknown mime")
	}
	if mimeFromExt("gif") != "image/gif" {
		t.Fatalf("gif mime")
	}
	if mimeFromExt("webp") != "image/webp" {
		t.Fatalf("webp mime")
	}
	if mimeFromExt("pdf") != "application/pdf" {
		t.Fatalf("pdf mime")
	}
	if mimeFromExt("txt") != "text/plain" {
		t.Fatalf("txt mime")
	}
	if mimeFromExt("json") != "application/json" {
		t.Fatalf("json mime")
	}
	if mimeFromExt("csv") != "text/csv" {
		t.Fatalf("csv mime")
	}
	if v, ok := asInt(int64(2)); !ok || v != 2 {
		t.Fatalf("asInt int64")
	}
	if v, ok := asInt(1); !ok || v != 1 {
		t.Fatalf("asInt int")
	}
	if _, ok := asInt("bad"); ok {
		t.Fatalf("asInt string")
	}
	if mapBudgetToEffort(512) != "low" {
		t.Fatalf("budget low")
	}
	if mapBudgetToEffort(2000) != "medium" {
		t.Fatalf("budget medium")
	}
	if mapGeminiRoleToCodex("model") != "assistant" {
		t.Fatalf("map role model")
	}
	if mapGeminiRoleToCodex("unknown") != "user" {
		t.Fatalf("map role unknown")
	}
}

func TestRegistryErrorPaths(t *testing.T) {
	r := NewRegistry()
	if got := r.GetTargetFormat(nil); got != "" {
		t.Fatalf("expected empty target format")
	}
	if _, err := r.TransformRequest(domain.ClientTypeOpenAI, domain.ClientType("bogus"), []byte("{}"), "m", false); err == nil {
		t.Fatalf("expected TransformRequest error")
	}
	if _, err := r.TransformResponse(domain.ClientTypeOpenAI, domain.ClientType("bogus"), []byte("{}")); err == nil {
		t.Fatalf("expected TransformResponse error")
	}
	if _, err := r.TransformStreamChunk(domain.ClientTypeOpenAI, domain.ClientType("bogus"), []byte(""), NewTransformState()); err == nil {
		t.Fatalf("expected TransformStreamChunk error")
	}
}

func TestToolNameMapCollision(t *testing.T) {
	long := strings.Repeat("a", maxToolNameLen+10)
	long2 := long
	m := buildShortNameMap([]string{long, long2})
	if m[long] == "" {
		t.Fatalf("short name missing")
	}
}

func TestSplitFunctionNameUnderscoreFallback(t *testing.T) {
	name, callID := splitFunctionName("tool_x")
	if name != "tool_x" || callID != "" {
		t.Fatalf("expected fallback for underscore")
	}
	name, callID = splitFunctionName("tool_call_1")
	if name != "tool" || callID != "call_1" {
		t.Fatalf("expected _call_ branch")
	}
}

func TestMapBudgetToEffortNegatives(t *testing.T) {
	if mapBudgetToEffort(-1) != "auto" {
		t.Fatalf("expected auto for -1")
	}
	if mapBudgetToEffort(-2) != "" {
		t.Fatalf("expected empty for other negatives")
	}
	if mapBudgetToEffort(0) != "none" {
		t.Fatalf("expected none for 0")
	}
}

func TestRegistrySameTypePassThrough(t *testing.T) {
	r := NewRegistry()
	body := []byte("abc")
	out, err := r.TransformRequest(domain.ClientTypeOpenAI, domain.ClientTypeOpenAI, body, "m", false)
	if err != nil || string(out) != "abc" {
		t.Fatalf("request passthrough failed")
	}
	out, err = r.TransformResponse(domain.ClientTypeOpenAI, domain.ClientTypeOpenAI, body)
	if err != nil || string(out) != "abc" {
		t.Fatalf("response passthrough failed")
	}
	out, err = r.TransformStreamChunk(domain.ClientTypeOpenAI, domain.ClientTypeOpenAI, body, NewTransformState())
	if err != nil || string(out) != "abc" {
		t.Fatalf("stream passthrough failed")
	}
}

func TestHasValidSignatureForFunctionCalls(t *testing.T) {
	if !hasValidSignatureForFunctionCalls(nil, strings.Repeat("a", MinSignatureLength)) {
		t.Fatalf("expected true for global signature")
	}
	if hasValidSignatureForFunctionCalls(nil, "") {
		t.Fatalf("expected false without signature")
	}
}

func TestParseInlineImageNonData(t *testing.T) {
	if parseInlineImage("http://example.com") != nil {
		t.Fatalf("expected nil for http")
	}
}

func TestHasWebSearchTool(t *testing.T) {
	tools := []ClaudeTool{{Type: "web_search_20250305"}, {Name: "google_search"}}
	if !hasWebSearchTool(tools) {
		t.Fatalf("expected web search tool")
	}
}

func TestHasWebSearchToolByName(t *testing.T) {
	tools := []ClaudeTool{{Name: "google_search"}}
	if !hasWebSearchTool(tools) {
		t.Fatalf("expected true for google_search")
	}
}

func TestCodexUserAgentInjectExtract(t *testing.T) {
	raw := []byte(`{"k":"v"}`)
	ua := "opencode/1.0"
	updated := InjectCodexUserAgent(raw, ua)
	if got := ExtractCodexUserAgent(updated); got != ua {
		t.Fatalf("expected user agent")
	}
	clean := StripCodexUserAgent(updated)
	if got := ExtractCodexUserAgent(clean); got != "" {
		t.Fatalf("expected stripped user agent")
	}
}

func TestCodexInstructionsForModelEnabled(t *testing.T) {
	SetCodexInstructionsEnabled(true)
	defer SetCodexInstructionsEnabled(false)
	instructions := CodexInstructionsForModel("gpt-5.3", "")
	if instructions == "" {
		t.Fatalf("expected instructions when enabled")
	}
	opencodeInstructions := CodexInstructionsForModel("gpt-5.3", "opencode/1.0")
	if opencodeInstructions == "" {
		t.Fatalf("expected opencode instructions")
	}
}

func TestHasValidSignatureForFunctionCallsInMessages(t *testing.T) {
	msgs := []ClaudeMessage{{
		Role:    "assistant",
		Content: []interface{}{map[string]interface{}{"type": "thinking", "signature": strings.Repeat("a", MinSignatureLength)}},
	}}
	if !hasValidSignatureForFunctionCalls(msgs, "") {
		t.Fatalf("expected true from message signature")
	}
	msgs = []ClaudeMessage{{
		Role:    "assistant",
		Content: []interface{}{map[string]interface{}{"type": "thinking", "signature": "short"}},
	}}
	if hasValidSignatureForFunctionCalls(msgs, "") {
		t.Fatalf("expected false for short signature")
	}
}

func TestRemapFunctionCallArgsPathsArray(t *testing.T) {
	args := map[string]interface{}{"query": "q", "paths": []interface{}{"/tmp"}}
	remapFunctionCallArgs("glob", args)
	if args["path"] != "/tmp" {
		t.Fatalf("expected path from paths")
	}
}

func TestParseFilePartMissingFields(t *testing.T) {
	if parseFilePart(map[string]interface{}{"file": map[string]interface{}{"filename": "a.txt"}}) != nil {
		t.Fatalf("expected nil for missing file_data")
	}
	if parseFilePart(map[string]interface{}{"file": map[string]interface{}{"file_data": "Zg=="}}) != nil {
		t.Fatalf("expected nil for missing filename")
	}
}

func TestHasWebSearchToolByNameRetrieval(t *testing.T) {
	tools := []ClaudeTool{{Name: "google_search_retrieval"}}
	if !hasWebSearchTool(tools) {
		t.Fatalf("expected true for google_search_retrieval")
	}
}

func TestExtractFirstPathString(t *testing.T) {
	args := map[string]interface{}{"paths": "./path"}
	remapFunctionCallArgs("grep", args)
	if args["path"] != "./path" {
		t.Fatalf("expected path from string")
	}
}

func TestShortenNameIfNeededNoChange(t *testing.T) {
	name := "short"
	if shortenNameIfNeeded(name) != name {
		t.Fatalf("expected unchanged")
	}
}

func TestHasValidSignatureForFunctionCallsNonAssistant(t *testing.T) {
	msgs := []ClaudeMessage{{Role: "user", Content: []interface{}{map[string]interface{}{"type": "thinking", "signature": strings.Repeat("a", MinSignatureLength)}}}}
	if hasValidSignatureForFunctionCalls(msgs, "") {
		t.Fatalf("expected false for non-assistant")
	}
}

func TestExtractFirstPathEmptyArray(t *testing.T) {
	if extractFirstPath([]interface{}{}) != "." {
		t.Fatalf("expected default path")
	}
}

func TestSplitFunctionNameCallUnderscoreBranch(t *testing.T) {
	name, callID := splitFunctionName("tool_call_1")
	if name != "tool" || callID != "call_1" {
		t.Fatalf("expected _call_ branch")
	}
}

func TestFilterInvalidThinkingBlocksAdditional(t *testing.T) {
	msgs := []ClaudeMessage{{
		Role:    "assistant",
		Content: []interface{}{map[string]interface{}{"type": "thinking", "thinking": "t", "signature": "short"}},
	}}
	FilterInvalidThinkingBlocks(msgs)
	raw, _ := json.Marshal(msgs)
	if !strings.Contains(string(raw), "text") {
		t.Fatalf("expected thinking downgraded to text")
	}
}

func TestFilterInvalidThinkingBlocksTrailingSignature(t *testing.T) {
	msgs := []ClaudeMessage{{
		Role:    "model",
		Content: []interface{}{map[string]interface{}{"type": "thinking", "thinking": "", "signature": "sig1234567"}},
	}}
	count := FilterInvalidThinkingBlocks(msgs)
	if count != 0 {
		t.Fatalf("expected no removal")
	}
}

func TestRegistryErrorMissingFromMap(t *testing.T) {
	r := NewRegistry()
	if _, err := r.TransformRequest(domain.ClientType("bogus"), domain.ClientTypeOpenAI, []byte("{}"), "m", false); err == nil {
		t.Fatalf("expected error for missing from")
	}
}

func TestHasFunctionCallsHelper(t *testing.T) {
	msgs := []ClaudeMessage{{Role: "assistant", Content: []interface{}{map[string]interface{}{"type": "tool_use"}}}}
	if !hasFunctionCalls(msgs) {
		t.Fatalf("expected function calls")
	}
}

func TestParseInlineImageInvalidBase64(t *testing.T) {
	if parseInlineImage("data:image/png;base64,!!!") != nil {
		t.Fatalf("expected nil")
	}
}

func TestFilterInvalidThinkingBlocksNonThinking(t *testing.T) {
	msgs := []ClaudeMessage{{
		Role:    "assistant",
		Content: []interface{}{map[string]interface{}{"type": "text", "text": "hi"}, "raw"},
	}}
	count := FilterInvalidThinkingBlocks(msgs)
	if count != 0 {
		t.Fatalf("expected no removal")
	}
}

func TestHelpers_ShortenNameIfNeededLong(t *testing.T) {
	name := strings.Repeat("a", maxToolNameLen+5)
	short := shortenNameIfNeeded(name)
	if len(short) > maxToolNameLen {
		t.Fatalf("expected shortened")
	}
	if short == name {
		t.Fatalf("expected shortened name to differ")
	}
}

func TestSplitFunctionNameCallMid(t *testing.T) {
	name, callID := splitFunctionName("tool_call_1_extra")
	if name != "tool" || callID != "call_1_extra" {
		t.Fatalf("unexpected split")
	}
}

func TestHasValidSignatureForFunctionCallsContentNotSlice(t *testing.T) {
	msgs := []ClaudeMessage{{Role: "assistant", Content: "text"}}
	if hasValidSignatureForFunctionCalls(msgs, "") {
		t.Fatalf("expected false for non-slice content")
	}
}

func TestHasFunctionCallsFalse(t *testing.T) {
	msgs := []ClaudeMessage{{Role: "assistant", Content: []interface{}{map[string]interface{}{"type": "text", "text": "hi"}}}}
	if hasFunctionCalls(msgs) {
		t.Fatalf("expected false")
	}
}

func TestSplitFunctionNameUnderscoreNoCall(t *testing.T) {
	name, callID := splitFunctionName("tool_x")
	if name != "tool_x" || callID != "" {
		t.Fatalf("expected no call id")
	}
}

func TestParseInlineImageMalformed(t *testing.T) {
	if parseInlineImage("data:image/png;base64") != nil {
		t.Fatalf("expected nil")
	}
}

func TestRegistryResponseStreamMissingFrom(t *testing.T) {
	r := NewRegistry()
	if _, err := r.TransformResponse(domain.ClientType("bogus"), domain.ClientTypeOpenAI, []byte("{}")); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := r.TransformStreamChunk(domain.ClientType("bogus"), domain.ClientTypeOpenAI, []byte(""), NewTransformState()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestHasFunctionCallsNonAssistant(t *testing.T) {
	msgs := []ClaudeMessage{{Role: "user", Content: []interface{}{map[string]interface{}{"type": "tool_use"}}}}
	if !hasFunctionCalls(msgs) {
		t.Fatalf("expected true regardless of role")
	}
}

func TestSplitFunctionNameNoCallSuffix(t *testing.T) {
	name, callID := splitFunctionName("tool_x_y")
	if name != "tool_x_y" || callID != "" {
		t.Fatalf("expected no call id")
	}
}

func TestHasFunctionCallsNonMapBlocks(t *testing.T) {
	msgs := []ClaudeMessage{{Role: "assistant", Content: []interface{}{"raw"}}}
	if hasFunctionCalls(msgs) {
		t.Fatalf("expected false for non-map blocks")
	}
}

func TestSplitFunctionNameLeadingCall(t *testing.T) {
	name, callID := splitFunctionName("_call_1")
	if name != "_call_1" || callID != "" {
		t.Fatalf("expected no split")
	}
}

func TestHasFunctionCallsMissingType(t *testing.T) {
	msgs := []ClaudeMessage{{Role: "assistant", Content: []interface{}{map[string]interface{}{"foo": "bar"}}}}
	if hasFunctionCalls(msgs) {
		t.Fatalf("expected false when no type")
	}
}

func TestHelpers_SplitFunctionNameVariants(t *testing.T) {
	base, suffix := splitFunctionName("tool_call_123")
	if base != "tool" || suffix != "call_123" {
		t.Fatalf("unexpected split: %q %q", base, suffix)
	}
	base, suffix = splitFunctionName("plain")
	if base != "plain" || suffix != "" {
		t.Fatalf("expected default split")
	}
}

func TestShortenNameIfNeededLong(t *testing.T) {
	name := strings.Repeat("a", maxToolNameLen+5)
	short := shortenNameIfNeeded(name)
	if len(short) > maxToolNameLen {
		t.Fatalf("expected shortened name")
	}
	if short == name {
		t.Fatalf("expected shortened name to differ")
	}
}

func TestHelpers_RemapFunctionCallArgs(t *testing.T) {
	remapFunctionCallArgs("grep", nil)

	grepArgs := map[string]interface{}{"query": "hi", "paths": []interface{}{"/tmp"}}
	remapFunctionCallArgs("grep", grepArgs)
	if _, ok := grepArgs["query"]; ok || grepArgs["pattern"] != "hi" {
		t.Fatalf("expected grep pattern")
	}
	if grepArgs["path"] != "/tmp" {
		t.Fatalf("expected grep path")
	}

	globArgs := map[string]interface{}{"query": "hi"}
	remapFunctionCallArgs("glob", globArgs)
	if globArgs["path"] != "." || globArgs["pattern"] != "hi" {
		t.Fatalf("expected glob defaults")
	}

	readArgs := map[string]interface{}{"path": "file.txt"}
	remapFunctionCallArgs("read", readArgs)
	if readArgs["file_path"] != "file.txt" {
		t.Fatalf("expected read file_path")
	}
	if _, ok := readArgs["path"]; ok {
		t.Fatalf("expected path removed")
	}

	lsArgs := map[string]interface{}{}
	remapFunctionCallArgs("ls", lsArgs)
	if lsArgs["path"] != "." {
		t.Fatalf("expected ls path")
	}
}

func TestHelpers_RemapFunctionCallArgsGrepDefaultPath(t *testing.T) {
	args := map[string]interface{}{"query": "hi"}
	remapFunctionCallArgs("grep", args)
	if args["path"] != "." {
		t.Fatalf("expected default path")
	}
}

func TestMimeFromExtMore(t *testing.T) {
	if mimeFromExt("csv") != "text/csv" {
		t.Fatalf("expected csv mime")
	}
}

func TestHelpers_MimeFromExtAll(t *testing.T) {
	cases := map[string]string{
		"png":  "image/png",
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"gif":  "image/gif",
		"webp": "image/webp",
		"pdf":  "application/pdf",
		"txt":  "text/plain",
		"json": "application/json",
		"csv":  "text/csv",
		"exe":  "",
	}
	for ext, want := range cases {
		if got := mimeFromExt(ext); got != want {
			t.Fatalf("unexpected mime for %s: %s", ext, got)
		}
	}
}

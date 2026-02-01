package converter

import "testing"

func TestRemapFunctionCallArgs(t *testing.T) {
	args := map[string]interface{}{"query": "foo", "paths": []interface{}{"a", "b"}}
	remapFunctionCallArgs("grep", args)
	if args["pattern"] != "foo" {
		t.Fatalf("expected pattern remap")
	}
	if args["path"] != "a" {
		t.Fatalf("expected path remap")
	}
	if _, ok := args["query"]; ok {
		t.Fatalf("expected query removed")
	}

	args = map[string]interface{}{"path": "x"}
	remapFunctionCallArgs("read", args)
	if args["file_path"] != "x" {
		t.Fatalf("expected file_path remap")
	}

	args = map[string]interface{}{}
	remapFunctionCallArgs("ls", args)
	if args["path"] != "." {
		t.Fatalf("expected default path")
	}
}

func TestExtractFirstPath(t *testing.T) {
	if p := extractFirstPath([]interface{}{"x"}); p != "x" {
		t.Fatalf("unexpected %q", p)
	}
	if p := extractFirstPath("y"); p != "y" {
		t.Fatalf("unexpected %q", p)
	}
	if p := extractFirstPath(123); p != "." {
		t.Fatalf("unexpected %q", p)
	}
}

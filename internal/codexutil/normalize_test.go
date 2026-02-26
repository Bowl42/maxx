package codexutil

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestNormalizeCodexInput_KeepRoleWhenTypeMissing(t *testing.T) {
	in := []byte(`{
		"input":[
			{"role":"user","content":"hello"}
		]
	}`)

	out := NormalizeCodexInput(in)
	if got := gjson.GetBytes(out, "input.0.role").String(); got != "user" {
		t.Fatalf("input.0.role = %q, want %q", got, "user")
	}
}

func TestNormalizeCodexInput_RemoveRoleForExplicitNonMessageType(t *testing.T) {
	in := []byte(`{
		"input":[
			{"type":"function_call","role":"assistant","name":"t","arguments":"{}"}
		]
	}`)

	out := NormalizeCodexInput(in)
	if gjson.GetBytes(out, "input.0.role").Exists() {
		t.Fatalf("expected role to be removed for non-message input")
	}
}


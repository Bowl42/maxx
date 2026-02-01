package converter

import (
	"testing"

	"github.com/awsl-project/maxx/internal/domain"
)

type dummyReq struct {
	out []byte
}

func (d *dummyReq) Transform(body []byte, _ string, _ bool) ([]byte, error) {
	return d.out, nil
}

type dummyResp struct {
	out []byte
}

func (d *dummyResp) Transform(body []byte) ([]byte, error) {
	return d.out, nil
}

func (d *dummyResp) TransformChunk(chunk []byte, _ *TransformState) ([]byte, error) {
	return append([]byte{}, chunk...), nil
}

func TestRegistryBasics(t *testing.T) {
	r := NewRegistry()
	req := &dummyReq{out: []byte("req")}
	resp := &dummyResp{out: []byte("resp")}
	r.Register(domain.ClientType("a"), domain.ClientType("b"), req, resp)

	if r.NeedConvert(domain.ClientType("a"), []domain.ClientType{domain.ClientType("a")}) {
		t.Fatalf("expected no convert")
	}
	if !r.NeedConvert(domain.ClientType("a"), []domain.ClientType{domain.ClientType("b")}) {
		t.Fatalf("expected convert")
	}
	if r.GetTargetFormat([]domain.ClientType{domain.ClientType("b"), domain.ClientType("c")}) != domain.ClientType("b") {
		t.Fatalf("unexpected target format")
	}

	out, err := r.TransformRequest(domain.ClientType("a"), domain.ClientType("b"), []byte("x"), "m", false)
	if err != nil || string(out) != "req" {
		t.Fatalf("unexpected transform request: %v %s", err, string(out))
	}
	out, err = r.TransformResponse(domain.ClientType("a"), domain.ClientType("b"), []byte("x"))
	if err != nil || string(out) != "resp" {
		t.Fatalf("unexpected transform response: %v %s", err, string(out))
	}
	out, err = r.TransformStreamChunk(domain.ClientType("a"), domain.ClientType("b"), []byte("chunk"), NewTransformState())
	if err != nil || string(out) != "chunk" {
		t.Fatalf("unexpected transform chunk: %v %s", err, string(out))
	}
}

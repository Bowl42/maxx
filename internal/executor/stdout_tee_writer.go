package executor

import (
	"io"
	"net/http"
	"os"

	"github.com/awsl-project/maxx/internal/debug"
)

// StdoutTeeResponseWriter writes response bytes to stdout before sending to the client.
type StdoutTeeResponseWriter struct {
	http.ResponseWriter
	stdout      io.Writer
	wrotePrefix bool
}

// NewStdoutTeeResponseWriter wraps an http.ResponseWriter with stdout teeing.
func NewStdoutTeeResponseWriter(w http.ResponseWriter) *StdoutTeeResponseWriter {
	return &StdoutTeeResponseWriter{
		ResponseWriter: w,
		stdout:         os.Stdout,
		wrotePrefix:    false,
	}
}

// Write outputs the bytes to stdout, then forwards to the underlying writer.
func (t *StdoutTeeResponseWriter) Write(b []byte) (int, error) {
	if len(b) > 0 && debug.Enabled() {
		if !t.wrotePrefix {
			_, _ = t.stdout.Write([]byte("=========返回字符串>>>>>>>>\n"))
			t.wrotePrefix = true
		}
		_, _ = t.stdout.Write(b)
	}
	return t.ResponseWriter.Write(b)
}

// WriteHeader forwards the status code.
func (t *StdoutTeeResponseWriter) WriteHeader(code int) {
	t.ResponseWriter.WriteHeader(code)
}

// Header forwards header access.
func (t *StdoutTeeResponseWriter) Header() http.Header {
	return t.ResponseWriter.Header()
}

// Flush forwards streaming flushes.
func (t *StdoutTeeResponseWriter) Flush() {
	if f, ok := t.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

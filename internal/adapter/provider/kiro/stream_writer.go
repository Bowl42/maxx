package kiro

import "io"

type teeWriter struct {
	primary io.Writer
	buffer  interface {
		WriteString(string) (int, error)
	}
}

func (tw *teeWriter) Write(p []byte) (int, error) {
	if tw.buffer != nil {
		_, _ = tw.buffer.WriteString(string(p))
	}
	return tw.primary.Write(p)
}

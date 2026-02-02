package debug

import "sync/atomic"

var enabled atomic.Bool

// SetEnabled sets whether debug logging is enabled.
func SetEnabled(v bool) {
	enabled.Store(v)
}

// Enabled reports whether debug logging is enabled.
func Enabled() bool {
	return enabled.Load()
}

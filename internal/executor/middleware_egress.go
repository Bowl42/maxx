package executor

import (
	"context"
	"time"

	"github.com/awsl-project/maxx/internal/flow"
)

func (e *Executor) egress(c *flow.Ctx) {
	state, ok := getExecState(c)
	if !ok {
		c.Next()
		return
	}

	c.Next()

	proxyReq := state.proxyReq
	if proxyReq != nil && proxyReq.Status == "IN_PROGRESS" {
		proxyReq.EndTime = time.Now()
		proxyReq.Duration = proxyReq.EndTime.Sub(proxyReq.StartTime)
		if state.ctx != nil && state.ctx.Err() != nil {
			proxyReq.Status = "CANCELLED"
			if state.ctx.Err() == context.Canceled {
				proxyReq.Error = "client disconnected"
			} else if state.ctx.Err() == context.DeadlineExceeded {
				proxyReq.Error = "request timeout"
			} else {
				proxyReq.Error = state.ctx.Err().Error()
			}
		} else {
			proxyReq.Status = "FAILED"
		}
		_ = e.proxyRequestRepo.Update(proxyReq)
		if e.broadcaster != nil {
			e.broadcaster.BroadcastProxyRequest(proxyReq)
		}
	}

	if state.currentAttempt != nil && state.currentAttempt.Status == "IN_PROGRESS" {
		state.currentAttempt.EndTime = time.Now()
		state.currentAttempt.Duration = state.currentAttempt.EndTime.Sub(state.currentAttempt.StartTime)
		if state.ctx != nil && state.ctx.Err() != nil {
			state.currentAttempt.Status = "CANCELLED"
		} else {
			state.currentAttempt.Status = "FAILED"
		}
		_ = e.attemptRepo.Update(state.currentAttempt)
		if e.broadcaster != nil {
			e.broadcaster.BroadcastProxyUpstreamAttempt(state.currentAttempt)
		}
	}

	_ = state.lastErr
}

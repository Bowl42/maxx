package flow

import (
	"io"
	"net/http"
)

type HandlerFunc func(*Ctx)

type Engine struct {
	handlers []HandlerFunc
}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Use(handlers ...HandlerFunc) {
	e.handlers = append(e.handlers, handlers...)
}

func (e *Engine) Handle(c *Ctx) {
	c.handlers = e.handlers
	c.index = -1
	c.Next()
}

func (e *Engine) HandleWith(c *Ctx, handlers ...HandlerFunc) {
	c.handlers = append(append([]HandlerFunc{}, e.handlers...), handlers...)
	c.index = -1
	c.Next()
}

type Ctx struct {
	Writer       http.ResponseWriter
	Request      *http.Request
	InboundBody  []byte
	OutboundBody []byte
	StreamBody   io.ReadCloser
	IsStream     bool
	Keys         map[string]interface{}
	Err          error

	handlers []HandlerFunc
	index    int
	aborted  bool
}

func NewCtx(w http.ResponseWriter, r *http.Request) *Ctx {
	return &Ctx{
		Writer:  w,
		Request: r,
		Keys:    make(map[string]interface{}),
	}
}

func (c *Ctx) Next() {
	if c.aborted {
		return
	}
	c.index++
	for c.index < len(c.handlers) {
		c.handlers[c.index](c)
		if c.aborted {
			return
		}
		c.index++
	}
}

func (c *Ctx) Abort() {
	c.aborted = true
}

func (c *Ctx) Set(key string, value interface{}) {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = value
}

func (c *Ctx) Get(key string) (interface{}, bool) {
	if c.Keys == nil {
		return nil, false
	}
	v, ok := c.Keys[key]
	return v, ok
}

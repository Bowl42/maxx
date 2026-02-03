package openai_to_codex

import (
	"github.com/awsl-project/maxx/internal/converter"
	"github.com/awsl-project/maxx/internal/domain"
)

func init() {
	converter.RegisterConverter(domain.ClientTypeOpenAI, domain.ClientTypeCodex, &Request{}, &Response{})
}

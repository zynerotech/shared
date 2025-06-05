package transport

import (
	"context"
)

type Handler interface {
	Handle(ctx context.Context, envelope Envelope) error
}

type ConsumerHandler struct {
	handler Handler
}

func NewConsumerHandler(handler Handler) *ConsumerHandler {
	return &ConsumerHandler{
		handler: handler,
	}
}

func (c *ConsumerHandler) Handle(ctx context.Context, msg Envelope) error {
	return c.handler.Handle(ctx, msg)
}

package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type Handler func(ctx context.Context, data []byte) error

type Subscriber struct {
	js nats.JetStreamContext
}

func NewSubscriber(nc *nats.Conn) (*Subscriber, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	return &Subscriber{js: js}, nil
}

// Subscribe creates a durable pull consumer and fans messages out to handler.
// Blocks until ctx is canceled. Errors from handler trigger nak + redelivery
// up to MaxDeliver; after that the message is parked.
func (s *Subscriber) Subscribe(ctx context.Context, stream, durable, subject string, h Handler) error {
	sub, err := s.js.PullSubscribe(subject, durable,
		nats.BindStream(stream),
		nats.AckWait(30*time.Second),
		nats.MaxDeliver(5),
	)
	if err != nil {
		return fmt.Errorf("pull subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		msgs, err := sub.Fetch(16, nats.MaxWait(2*time.Second))
		if err != nil {
			if err == nats.ErrTimeout {
				continue
			}
			return fmt.Errorf("fetch: %w", err)
		}

		for _, m := range msgs {
			if err := h(ctx, m.Data); err != nil {
				_ = m.Nak()
				continue
			}
			_ = m.Ack()
		}
	}
}

// DecodeInto is a small helper so handlers don't repeat the json boilerplate.
func DecodeInto(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("decode event: %w", err)
	}
	return nil
}

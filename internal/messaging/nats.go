package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Publisher publishes events to NATS JetStream. Streams must exist beforehand
// (created in EnsureStream during bootstrap).
type Publisher struct {
	js nats.JetStreamContext
}

func NewPublisher(nc *nats.Conn) (*Publisher, error) {
	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		return nil, fmt.Errorf("init jetstream: %w", err)
	}
	return &Publisher{js: js}, nil
}

// Publish marshals v to JSON and publishes synchronously. Returns once the
// server acks. Caller passes ctx for cancellation.
func (p *Publisher) Publish(ctx context.Context, subject string, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	msg := &nats.Msg{
		Subject: subject,
		Data:    payload,
		Header:  nats.Header{},
	}
	msg.Header.Set("Nats-Msg-Id", fmt.Sprintf("%s-%d", subject, time.Now().UnixNano()))
	msg.Header.Set("content-type", "application/json")

	ack, err := p.js.PublishMsg(msg, nats.Context(ctx))
	if err != nil {
		return fmt.Errorf("publish %s: %w", subject, err)
	}
	_ = ack // stream/sequence available if we ever need it
	return nil
}

// EnsureStream is idempotent — call it from main() on startup.
func EnsureStream(nc *nats.Conn, name string, subjects ...string) error {
	js, err := nc.JetStream()
	if err != nil {
		return err
	}

	cfg := &nats.StreamConfig{
		Name:       name,
		Subjects:   subjects,
		Retention:  nats.LimitsPolicy,
		MaxAge:     7 * 24 * time.Hour,
		Storage:    nats.FileStorage,
		Replicas:   1,
		Duplicates: 2 * time.Minute,
	}

	if _, err := js.StreamInfo(name); err != nil {
		if _, err := js.AddStream(cfg); err != nil {
			return fmt.Errorf("add stream %s: %w", name, err)
		}
		return nil
	}

	if _, err := js.UpdateStream(cfg); err != nil {
		return fmt.Errorf("update stream %s: %w", name, err)
	}
	return nil
}

// Package sender abstracts the actual delivery channel so we can swap
// providers without touching the subscriber wiring.
package sender

import (
	"context"

	"github.com/dimasd-angga/go-microkit/internal/logger"
)

type Channel string

const (
	ChannelEmail   Channel = "email"
	ChannelPush    Channel = "push"
	ChannelWebhook Channel = "webhook"
)

type Message struct {
	To      string
	Channel Channel
	Subject string
	Body    string
}

type Sender interface {
	Send(ctx context.Context, m Message) error
}

// Stdout is a no-op sender that prints to logs. Replaces real providers in
// dev; swap for SES / Twilio / FCM in production.
type Stdout struct{}

func (Stdout) Send(ctx context.Context, m Message) error {
	logger.Info(ctx, "sending notification",
		"channel", m.Channel,
		"to", m.To,
		"subject", m.Subject,
	)
	return nil
}

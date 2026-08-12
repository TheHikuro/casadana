package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

type Mailer struct {
	client      *resend.Client
	from        string
	adminNotify string
}

func NewMailer(apiKey, from, adminNotify string) *Mailer {
	return &Mailer{
		client:      resend.NewClient(apiKey),
		from:        from,
		adminNotify: adminNotify,
	}
}

type Message struct {
	To      string
	Subject string
	HTML    string
	// Text is the plain-text alternative. Sending it alongside the HTML part is
	// what keeps transactional mail out of spam folders, and it is the only
	// thing some clients will render.
	Text string
	// ReplyTo makes a reply reach a human: the owners on guest mail, the guest
	// on the owners' notification.
	ReplyTo string
	// IdempotencyKey lets Resend collapse a retry of the same notification into
	// a single delivery. Resend honours a key for 24h.
	IdempotencyKey string
}

func (m *Mailer) Send(ctx context.Context, msg Message) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("email: mailer not configured")
	}
	req := &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{msg.To},
		Subject: msg.Subject,
		Html:    msg.HTML,
		Text:    msg.Text,
		ReplyTo: msg.ReplyTo,
	}
	var err error
	if msg.IdempotencyKey != "" {
		_, err = m.client.Emails.SendWithOptions(ctx, req, &resend.SendEmailOptions{
			IdempotencyKey: msg.IdempotencyKey,
		})
	} else {
		_, err = m.client.Emails.SendWithContext(ctx, req)
	}
	if err != nil {
		return fmt.Errorf("email: send: %w", err)
	}
	return nil
}

func (m *Mailer) AdminNotifyAddress() string { return m.adminNotify }

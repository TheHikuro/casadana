package email

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// BookingData is the projection of a booking that the emails need. This package
// deliberately does not import the booking domain: it renders and sends
// messages, it knows nothing about the booking lifecycle.
type BookingData struct {
	ID         string
	VillaName  string
	GuestName  string
	GuestEmail string
	GuestPhone string
	CheckIn    time.Time
	CheckOut   time.Time
	Adults     int
	Children   int
	Message    string
	// Locale the guest browsed the site in. Empty falls back to DefaultLocale.
	Locale Locale
}

// nights counts the billed nights. Check-in and check-out are calendar dates at
// UTC midnight, so a plain hour division is exact here.
func (d BookingData) nights() int {
	n := int(d.CheckOut.Sub(d.CheckIn).Hours() / 24)
	if n < 0 {
		return 0
	}
	return n
}

func (d BookingData) locale() Locale {
	if d.Locale == "" {
		return DefaultLocale
	}
	return NormalizeLocale(string(d.Locale))
}

func (d BookingData) guestsLine(loc Locale) string {
	parts := []string{plural(loc, "guests.adult", d.Adults)}
	if d.Children > 0 {
		parts = append(parts, plural(loc, "guests.child", d.Children))
	}
	return strings.Join(parts, ", ")
}

// stayDetails is the block every guest email repeats, so a guest can check the
// dates without digging up the original request.
func (d BookingData) stayDetails(loc Locale) []detail {
	return []detail{
		{Label: t(loc, "label.villa"), Value: d.VillaName},
		{Label: t(loc, "label.checkin"), Value: formatDate(d.CheckIn, loc)},
		{Label: t(loc, "label.checkout"), Value: formatDate(d.CheckOut, loc)},
		{Label: t(loc, "label.nights"), Value: fmt.Sprintf("%d", d.nights())},
		{Label: t(loc, "label.guests"), Value: d.guestsLine(loc)},
	}
}

func (d BookingData) base(loc Locale) content {
	return content{
		Locale:     loc,
		Tagline:    t(loc, "brand.tagline"),
		SignedName: t(loc, "footer.signed"),
		SignedRole: t(loc, "footer.role"),
		ReplyHint:  t(loc, "footer.reply"),
	}
}

// The content builders below are pure: they turn a booking into the copy of one
// email and touch nothing else, which is what makes every email in this package
// assertable without a network round-trip.

func (d BookingData) receivedContent() content {
	loc := d.locale()
	c := d.base(loc)
	c.Subject = tf(loc, "received.subject", d.VillaName)
	c.Heading = t(loc, "received.heading")
	c.Greeting = tf(loc, "greeting", d.GuestName)
	c.Paragraphs = []string{tf(loc, "received.p1", d.VillaName), t(loc, "received.p2")}
	c.Details = d.stayDetails(loc)
	c.Note = t(loc, "received.note")
	return c
}

func (d BookingData) approvedContent() content {
	loc := d.locale()
	c := d.base(loc)
	c.Subject = tf(loc, "approved.subject", d.VillaName)
	c.Heading = t(loc, "approved.heading")
	c.Greeting = tf(loc, "greeting", d.GuestName)
	c.Paragraphs = []string{
		tf(loc, "approved.p1", d.VillaName, formatDate(d.CheckIn, loc), formatDate(d.CheckOut, loc)),
		t(loc, "approved.p2"),
	}
	c.Details = d.stayDetails(loc)
	c.Note = t(loc, "approved.note")
	return c
}

func (d BookingData) rejectedContent() content {
	loc := d.locale()
	c := d.base(loc)
	c.Subject = tf(loc, "rejected.subject", d.VillaName)
	c.Heading = t(loc, "rejected.heading")
	c.Greeting = tf(loc, "greeting", d.GuestName)
	c.Paragraphs = []string{
		tf(loc, "rejected.p1", d.VillaName, formatDate(d.CheckIn, loc), formatDate(d.CheckOut, loc)),
		t(loc, "rejected.p2"),
	}
	return c
}

func (d BookingData) cancelledContent() content {
	loc := d.locale()
	c := d.base(loc)
	c.Subject = tf(loc, "cancelled.subject", d.VillaName)
	c.Heading = t(loc, "cancelled.heading")
	c.Greeting = tf(loc, "greeting", d.GuestName)
	c.Paragraphs = []string{
		tf(loc, "cancelled.p1", d.VillaName, formatDate(d.CheckIn, loc), formatDate(d.CheckOut, loc)),
		t(loc, "cancelled.p2"),
	}
	c.Details = d.stayDetails(loc)
	return c
}

// ownerContent is always written in the owners' own language, whatever locale
// the guest browsed in, and repeats the guest's contact details so the owners
// can act on the request straight from their inbox.
func (d BookingData) ownerContent() content {
	loc := DefaultLocale
	c := d.base(loc)
	c.Subject = tf(loc, "owner.subject", d.VillaName, formatDate(d.CheckIn, loc), formatDate(d.CheckOut, loc))
	c.Heading = t(loc, "owner.heading")
	c.Paragraphs = []string{t(loc, "owner.p1"), t(loc, "owner.p2")}

	message := strings.TrimSpace(d.Message)
	if message == "" {
		message = t(loc, "owner.nomessage")
	}
	c.Details = append(d.stayDetails(loc),
		detail{Label: t(loc, "label.name"), Value: d.GuestName},
		detail{Label: t(loc, "label.email"), Value: d.GuestEmail},
		detail{Label: t(loc, "label.phone"), Value: d.GuestPhone},
		detail{Label: t(loc, "label.message"), Value: message},
		detail{Label: t(loc, "label.ref"), Value: d.ID},
	)
	return c
}

// SendGuestRequestReceived acknowledges a freshly submitted request.
func (m *Mailer) SendGuestRequestReceived(ctx context.Context, d BookingData) error {
	return m.sendGuest(ctx, d, "received", d.receivedContent())
}

// SendGuestApproved tells the guest the stay is confirmed.
func (m *Mailer) SendGuestApproved(ctx context.Context, d BookingData) error {
	return m.sendGuest(ctx, d, "approved", d.approvedContent())
}

// SendGuestRejected declines a request without leaving the guest waiting.
func (m *Mailer) SendGuestRejected(ctx context.Context, d BookingData) error {
	return m.sendGuest(ctx, d, "rejected", d.rejectedContent())
}

// SendGuestCancelled confirms a cancellation, whoever triggered it.
func (m *Mailer) SendGuestCancelled(ctx context.Context, d BookingData) error {
	return m.sendGuest(ctx, d, "cancelled", d.cancelledContent())
}

// SendOwnerNewRequest notifies the owners that a request is waiting.
func (m *Mailer) SendOwnerNewRequest(ctx context.Context, d BookingData) error {
	// Reply-To is the guest, so answering the notification answers the guest.
	msg, err := buildMessage(m.adminNotify, d.GuestEmail, "owner-request", d.ID, d.ownerContent())
	if err != nil {
		return err
	}
	return m.Send(ctx, msg)
}

// sendGuest addresses the guest and points Reply-To at the owners, so a guest
// replying to any of these emails reaches a human.
func (m *Mailer) sendGuest(ctx context.Context, d BookingData, kind string, c content) error {
	msg, err := buildMessage(d.GuestEmail, m.adminNotify, kind, d.ID, c)
	if err != nil {
		return err
	}
	return m.Send(ctx, msg)
}

// buildMessage renders the layout into a sendable message. The idempotency key
// is derived from the booking and the kind of email, so retrying the same
// notification can never deliver it twice.
func buildMessage(to, replyTo, kind, bookingID string, c content) (Message, error) {
	if len(c.Paragraphs) > 0 {
		c.Preheader = c.Paragraphs[0]
	}
	html, text, err := c.render()
	if err != nil {
		return Message{}, err
	}
	return Message{
		To:             to,
		ReplyTo:        replyTo,
		Subject:        c.Subject,
		HTML:           html,
		Text:           text,
		IdempotencyKey: fmt.Sprintf("booking/%s/%s", bookingID, kind),
	}, nil
}

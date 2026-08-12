package booking

import (
	"context"

	"github.com/TheHikuro/casadana/internal/platform/email"
	"github.com/TheHikuro/casadana/internal/villaslug"
)

// ResendMailer adapts the booking domain onto the transactional email package.
// The copy, the layout and the localisation all live in that package; this type
// only projects a Booking onto what an email needs — which is why no HTML
// appears anywhere in this file.
type ResendMailer struct {
	inner *email.Mailer
}

func NewResendMailer(m *email.Mailer) Mailer { return &ResendMailer{inner: m} }

// mailData projects a booking. The villa is passed by display name, never by
// slug: "casadana" is an internal identifier and has no business appearing in
// something a guest reads.
func mailData(b *Booking) email.BookingData {
	return email.BookingData{
		ID:         b.ID,
		VillaName:  villaslug.DisplayName(b.VillaSlug),
		GuestName:  b.GuestName,
		GuestEmail: b.GuestEmail,
		GuestPhone: b.GuestPhone,
		CheckIn:    b.CheckIn,
		CheckOut:   b.CheckOut,
		Adults:     b.Adults,
		Children:   b.Children,
		Message:    b.Message,
		// The locale the guest browsed in, captured at request time. Guest mail
		// sent weeks later — an approval, a cancellation — is still written in
		// their language because of it.
		Locale: email.Locale(b.Locale),
	}
}

func (r *ResendMailer) SendRequestReceived(ctx context.Context, b *Booking) error {
	return r.inner.SendGuestRequestReceived(ctx, mailData(b))
}

func (r *ResendMailer) SendOwnerNewRequest(ctx context.Context, b *Booking) error {
	return r.inner.SendOwnerNewRequest(ctx, mailData(b))
}

func (r *ResendMailer) SendApproved(ctx context.Context, b *Booking) error {
	return r.inner.SendGuestApproved(ctx, mailData(b))
}

func (r *ResendMailer) SendRejected(ctx context.Context, b *Booking) error {
	return r.inner.SendGuestRejected(ctx, mailData(b))
}

func (r *ResendMailer) SendCancelled(ctx context.Context, b *Booking) error {
	return r.inner.SendGuestCancelled(ctx, mailData(b))
}

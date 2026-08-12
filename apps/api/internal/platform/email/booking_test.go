package email

import (
	"strings"
	"testing"
	"time"
)

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func sample(loc Locale) BookingData {
	return BookingData{
		ID:         "b-1234",
		VillaName:  "Casa DaNa",
		GuestName:  "Jane Doe",
		GuestEmail: "jane@example.com",
		GuestPhone: "+33 6 12 34 56 78",
		CheckIn:    date("2026-07-01"),
		CheckOut:   date("2026-07-08"),
		Adults:     2,
		Children:   1,
		Message:    "Nous arriverons tard le soir.",
		Locale:     loc,
	}
}

func allContents(d BookingData) map[string]content {
	return map[string]content{
		"received":  d.receivedContent(),
		"approved":  d.approvedContent(),
		"rejected":  d.rejectedContent(),
		"cancelled": d.cancelledContent(),
		"owner":     d.ownerContent(),
	}
}

// No email may ship with an unfilled placeholder, an empty subject or a subject
// that forgot which villa it is about — in any of the three languages.
func TestContents_AreCompleteInEveryLocale(t *testing.T) {
	for _, loc := range []Locale{LocaleFR, LocaleEN, LocaleES} {
		d := sample(loc)
		for kind, c := range allContents(d) {
			if c.Subject == "" {
				t.Errorf("%s/%s: empty subject", loc, kind)
			}
			if !strings.Contains(c.Subject, d.VillaName) {
				t.Errorf("%s/%s: subject %q does not name the villa", loc, kind, c.Subject)
			}
			if c.Heading == "" {
				t.Errorf("%s/%s: empty heading", loc, kind)
			}
			if len(c.Paragraphs) == 0 {
				t.Errorf("%s/%s: no body copy", loc, kind)
			}
			for _, s := range append([]string{c.Subject, c.Heading, c.Greeting, c.Note}, c.Paragraphs...) {
				if strings.Contains(s, "%!") || strings.Contains(s, "%s") || strings.Contains(s, "%d") {
					t.Errorf("%s/%s: unfilled placeholder in %q", loc, kind, s)
				}
			}
		}
	}
}

// The guest-facing emails must be in the guest's language, not the owners'.
func TestContents_FollowTheGuestLocale(t *testing.T) {
	fr := sample(LocaleFR).approvedContent()
	en := sample(LocaleEN).approvedContent()
	es := sample(LocaleES).approvedContent()

	if fr.Subject == en.Subject || en.Subject == es.Subject {
		t.Errorf("subjects did not change with the locale: fr=%q en=%q es=%q", fr.Subject, en.Subject, es.Subject)
	}
	if !strings.HasPrefix(en.Greeting, "Hello") {
		t.Errorf("en greeting = %q, want an English greeting", en.Greeting)
	}
}

// The owners read their notification in their own language even when the guest
// browsed in another one.
func TestOwnerContent_StaysInTheOwnersLanguage(t *testing.T) {
	es := sample(LocaleES).ownerContent()
	fr := sample(LocaleFR).ownerContent()
	if es.Subject != fr.Subject {
		t.Errorf("owner subject follows the guest locale: es=%q fr=%q", es.Subject, fr.Subject)
	}
	if es.Locale != DefaultLocale {
		t.Errorf("owner locale = %q, want %q", es.Locale, DefaultLocale)
	}
}

// A guest reply must reach the owners, and an owner reply must reach the guest.
func TestBuildMessage_AddressingAndIdempotency(t *testing.T) {
	d := sample(LocaleFR)
	const owner = "hosts@casa-dana.com"

	guest, err := buildMessage(d.GuestEmail, owner, "approved", d.ID, d.approvedContent())
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	if guest.To != d.GuestEmail || guest.ReplyTo != owner {
		t.Errorf("guest mail addressed To=%q ReplyTo=%q, want To=%q ReplyTo=%q",
			guest.To, guest.ReplyTo, d.GuestEmail, owner)
	}
	if guest.IdempotencyKey != "booking/b-1234/approved" {
		t.Errorf("idempotency key = %q", guest.IdempotencyKey)
	}

	notice, err := buildMessage(owner, d.GuestEmail, "owner-request", d.ID, d.ownerContent())
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	if notice.To != owner || notice.ReplyTo != d.GuestEmail {
		t.Errorf("owner mail addressed To=%q ReplyTo=%q, want To=%q ReplyTo=%q",
			notice.To, notice.ReplyTo, owner, d.GuestEmail)
	}
	if guest.IdempotencyKey == notice.IdempotencyKey {
		t.Error("two different emails about the same booking share an idempotency key")
	}
}

// Both parts must be produced, and the HTML one must escape guest-supplied
// text: a guest called `<script>` is a guest, not markup.
func TestBuildMessage_RendersBothPartsAndEscapes(t *testing.T) {
	d := sample(LocaleFR)
	d.GuestName = `Jane <script>alert("x")</script>`

	msg, err := buildMessage(d.GuestEmail, "hosts@casa-dana.com", "received", d.ID, d.receivedContent())
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	if msg.HTML == "" || msg.Text == "" {
		t.Fatalf("missing part: html=%d bytes text=%d bytes", len(msg.HTML), len(msg.Text))
	}
	if strings.Contains(msg.HTML, "<script>") {
		t.Error("guest name was injected into the HTML unescaped")
	}
	if !strings.Contains(msg.HTML, "&lt;script&gt;") {
		t.Error("guest name was not escaped into the HTML part")
	}
	for _, want := range []string{"Casa DaNa", "01/07/2026", "08/07/2026"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("text part is missing %q", want)
		}
	}
}

func TestStayDetails_NightsAndGuests(t *testing.T) {
	d := sample(LocaleFR)
	if got := d.nights(); got != 7 {
		t.Errorf("nights = %d, want 7", got)
	}
	if got := d.guestsLine(LocaleFR); got != "2 adultes, 1 enfant" {
		t.Errorf("guests = %q", got)
	}

	solo := sample(LocaleFR)
	solo.Adults, solo.Children = 1, 0
	if got := solo.guestsLine(LocaleFR); got != "1 adulte" {
		t.Errorf("solo guests = %q, want singular and no children", got)
	}

	// A same-day range is nonsense but must not render a negative night count.
	odd := sample(LocaleFR)
	odd.CheckOut = odd.CheckIn
	if got := odd.nights(); got != 0 {
		t.Errorf("nights for an empty range = %d, want 0", got)
	}
}

package audit

import (
	"strings"
	"testing"
)

func TestNewEvent_Validation(t *testing.T) {
	tests := []struct {
		name    string
		in      NewEventInput
		wantErr bool
	}{
		{"valid", NewEventInput{VillaSlug: "casadana", Type: TypePricing, Message: "price updated"}, false},
		{"unknown villa", NewEventInput{VillaSlug: "ghost", Type: TypePricing, Message: "x"}, true},
		{"empty villa", NewEventInput{Type: TypePricing, Message: "x"}, true},
		{"unknown type", NewEventInput{VillaSlug: "casadana", Type: "nope", Message: "x"}, true},
		{"empty type", NewEventInput{VillaSlug: "casadana", Message: "x"}, true},
		{"empty message", NewEventInput{VillaSlug: "casadana", Type: TypeReview}, true},
		{"blank message", NewEventInput{VillaSlug: "casadana", Type: TypeReview, Message: "   "}, true},
		{"message at limit", NewEventInput{VillaSlug: "casadana", Type: TypeReview, Message: strings.Repeat("a", 500)}, false},
		{"message too long", NewEventInput{VillaSlug: "casadana", Type: TypeReview, Message: strings.Repeat("a", 501)}, true},
		{"actor optional", NewEventInput{VillaSlug: "casacasay", Type: TypeSystem, Message: "boot"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := NewEvent(tt.in)
			if tt.wantErr {
				if err != ErrInvalidPayload {
					t.Fatalf("err = %v, want ErrInvalidPayload", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewEvent: %v", err)
			}
			if e.ID == "" {
				t.Error("ID is empty")
			}
		})
	}
}

func TestNewEvent_TrimsMessageAndActor(t *testing.T) {
	e, err := NewEvent(NewEventInput{
		VillaSlug:  "casadana",
		Type:       TypeOwner,
		Message:    "  owner changed  ",
		ActorEmail: "  admin@casadana.fr ",
	})
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	if e.Message != "owner changed" {
		t.Errorf("Message = %q, want %q", e.Message, "owner changed")
	}
	if e.ActorEmail != "admin@casadana.fr" {
		t.Errorf("ActorEmail = %q, want %q", e.ActorEmail, "admin@casadana.fr")
	}
}

func TestParseEventType(t *testing.T) {
	tests := []struct {
		in      string
		want    EventType
		wantErr bool
	}{
		{"reservation", TypeReservation, false},
		{"pricing", TypePricing, false},
		{"review", TypeReview, false},
		{"owner", TypeOwner, false},
		{"system", TypeSystem, false},
		{" pricing ", TypePricing, false},
		{"Pricing", "", true},
		{"booking", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseEventType(tt.in)
			if tt.wantErr {
				if err != ErrInvalidPayload {
					t.Fatalf("err = %v, want ErrInvalidPayload", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseEventType: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizePaging(t *testing.T) {
	tests := []struct {
		name                            string
		page, limit                     int
		wantPage, wantLimit, wantOffset int
	}{
		{"defaults", 0, 0, 1, 20, 0},
		{"page zero floors to 1", 0, 10, 1, 10, 0},
		{"negative page floors to 1", -5, 10, 1, 10, 0},
		{"second page", 2, 20, 2, 20, 20},
		{"third page of 10", 3, 10, 3, 10, 20},
		{"negative limit defaults", 2, -1, 2, 20, 20},
		{"limit clamped to max", 2, 500, 2, 100, 100},
		{"limit at max kept", 1, 100, 1, 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, limit, offset := normalizePaging(tt.page, tt.limit)
			if page != tt.wantPage || limit != tt.wantLimit || offset != tt.wantOffset {
				t.Errorf("got (%d, %d, %d), want (%d, %d, %d)",
					page, limit, offset, tt.wantPage, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}

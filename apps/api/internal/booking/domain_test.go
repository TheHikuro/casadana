package booking

import (
	"strings"
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func validCmd() NewBookingInput {
	return NewBookingInput{
		VillaSlug:  "casadana",
		GuestName:  "Jane Doe",
		GuestEmail: "jane@example.com",
		GuestPhone: "+33123456789",
		CheckIn:    d("2026-07-01"),
		CheckOut:   d("2026-07-08"),
		Adults:     2,
		Children:   1,
		Message:    "Looking forward to it",
		Now:        d("2026-05-12"),
	}
}

func TestNewBooking_Happy(t *testing.T) {
	b, err := NewBooking(validCmd())
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	if b.Status != StatusPending {
		t.Errorf("status = %s, want pending", b.Status)
	}
	if b.ID == "" {
		t.Error("ID was empty")
	}
}

func TestNewBooking_Rejects(t *testing.T) {
	cases := map[string]func(*NewBookingInput){
		"check_out before check_in": func(c *NewBookingInput) { c.CheckOut = c.CheckIn.AddDate(0, 0, -1) },
		"check_out equals check_in": func(c *NewBookingInput) { c.CheckOut = c.CheckIn },
		"check_in in the past":      func(c *NewBookingInput) { c.CheckIn = c.Now.AddDate(0, 0, -1); c.CheckOut = c.Now },
		"adults < 1":                func(c *NewBookingInput) { c.Adults = 0 },
		"children < 0":              func(c *NewBookingInput) { c.Children = -1 },
		"empty guest name":          func(c *NewBookingInput) { c.GuestName = strings.Repeat(" ", 3) },
		"empty guest email":         func(c *NewBookingInput) { c.GuestEmail = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := validCmd()
			mutate(&c)
			if _, err := NewBooking(c); err == nil {
				t.Fatalf("expected error for %q, got nil", name)
			}
		})
	}
}

func TestNewBooking_DefaultsSourceToDirect(t *testing.T) {
	c := validCmd()
	c.Source = ""
	b, err := NewBooking(c)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	if b.Source != "direct" {
		t.Errorf("Source = %q, want direct", b.Source)
	}
}

func TestNewBooking_KeepsExplicitSource(t *testing.T) {
	c := validCmd()
	c.Source = "airbnb"
	b, err := NewBooking(c)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	if b.Source != "airbnb" {
		t.Errorf("Source = %q, want airbnb", b.Source)
	}
}

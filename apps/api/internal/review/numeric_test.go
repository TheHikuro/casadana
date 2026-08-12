package review

import (
	"math"
	"math/big"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestNumericRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   float64
	}{
		{name: "integer score", in: 5},
		{name: "one decimal", in: 4.7},
		{name: "two decimals", in: 4.85},
		{name: "leading zero", in: 0.5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := numericToScore(numericFromScore(&tc.in))
			if got == nil {
				t.Fatalf("round trip of %v = nil", tc.in)
			}
			if *got != tc.in {
				t.Errorf("round trip = %v, want %v", *got, tc.in)
			}
		})
	}
}

func TestNumericToScore_Edges(t *testing.T) {
	tests := []struct {
		name string
		in   pgtype.Numeric
		want *float64
	}{
		{name: "null reads as no score", in: pgtype.Numeric{}},
		{name: "NaN reads as no score", in: pgtype.Numeric{NaN: true, Valid: true}},
		{name: "infinity reads as no score", in: pgtype.Numeric{InfinityModifier: pgtype.Infinity, Valid: true}},
		{name: "nil Int reads as no score", in: pgtype.Numeric{Valid: true}},
		{name: "scaled int", in: pgtype.Numeric{Int: big.NewInt(487), Exp: -2, Valid: true}, want: ptr(4.87)},
		// An AVG comes back with a long tail; the score is what gets displayed.
		{name: "long average is rounded", in: pgtype.Numeric{Int: big.NewInt(4866666), Exp: -6, Valid: true}, want: ptr(4.87)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := numericToScore(tc.in)
			switch {
			case tc.want == nil && got != nil:
				t.Errorf("numericToScore = %v, want nil", *got)
			case tc.want != nil && got == nil:
				t.Errorf("numericToScore = nil, want %v", *tc.want)
			case tc.want != nil && *got != *tc.want:
				t.Errorf("numericToScore = %v, want %v", *got, *tc.want)
			}
		})
	}
}

// An unscored category is stored as NULL, and a NaN or infinite score has no
// decimal form the NUMERIC column would take either.
func TestNumericFromScore_NilAndNonFiniteAreNull(t *testing.T) {
	if n := numericFromScore(nil); n.Valid {
		t.Error("numericFromScore(nil) valid = true, want NULL")
	}
	for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if n := numericFromScore(&f); n.Valid {
			t.Errorf("numericFromScore(%v) valid = true, want NULL", f)
		}
	}
}

// The decimal text form is what keeps 4.7 from storing as 4.699999...
func TestNumericFromScore_KeepsDecimalForm(t *testing.T) {
	n := numericFromScore(ptr(4.7))
	if !n.Valid {
		t.Fatal("numericFromScore(4.7) is NULL")
	}
	if n.Int.String() != "47" || n.Exp != -1 {
		t.Errorf("Int/Exp = %s/%d, want 47/-1", n.Int, n.Exp)
	}
}

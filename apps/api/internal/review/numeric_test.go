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
		{name: "zero", in: 0},
		{name: "integer score", in: 5},
		{name: "one decimal", in: 4.7},
		{name: "two decimals", in: 4.85},
		{name: "leading zero", in: 0.5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := numericToFloat(numericFromFloat(tc.in))
			if got != tc.in {
				t.Errorf("round trip = %v, want %v", got, tc.in)
			}
		})
	}
}

func TestNumericToFloat_Edges(t *testing.T) {
	tests := []struct {
		name string
		in   pgtype.Numeric
		want float64
	}{
		{name: "null reads as zero", in: pgtype.Numeric{}, want: 0},
		{name: "NaN reads as zero", in: pgtype.Numeric{NaN: true, Valid: true}, want: 0},
		{name: "infinity reads as zero", in: pgtype.Numeric{InfinityModifier: pgtype.Infinity, Valid: true}, want: 0},
		{name: "nil Int reads as zero", in: pgtype.Numeric{Valid: true}, want: 0},
		{name: "scaled int", in: pgtype.Numeric{Int: big.NewInt(487), Exp: -2, Valid: true}, want: 4.87},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := numericToFloat(tc.in); got != tc.want {
				t.Errorf("numericToFloat = %v, want %v", got, tc.want)
			}
		})
	}
}

// A NaN or infinite score has no decimal form: store NULL rather than a value
// the NUMERIC column would reject or a reader would misread.
func TestNumericFromFloat_NonFiniteIsNull(t *testing.T) {
	for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if n := numericFromFloat(f); n.Valid {
			t.Errorf("numericFromFloat(%v) valid = true, want NULL", f)
		}
	}
}

// The decimal text form is what keeps 4.7 from storing as 4.699999...
func TestNumericFromFloat_KeepsDecimalForm(t *testing.T) {
	n := numericFromFloat(4.7)
	if !n.Valid {
		t.Fatal("numericFromFloat(4.7) is NULL")
	}
	if n.Int.String() != "47" || n.Exp != -1 {
		t.Errorf("Int/Exp = %s/%d, want 47/-1", n.Int, n.Exp)
	}
}

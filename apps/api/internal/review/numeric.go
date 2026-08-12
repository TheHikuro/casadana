package review

import (
	"math"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// numericToScore reads a NUMERIC column as an optional score. NULL, NaN and
// infinities all read as "no score": the difference between a category nobody
// rated and one rated zero is exactly what the public bars need in order to
// leave the unrated ones off.
func numericToScore(n pgtype.Numeric) *float64 {
	// A valid Numeric with no mantissa is malformed rather than zero, and
	// "unrated" is the honest reading of it.
	if !n.Valid || n.NaN || n.Int == nil {
		return nil
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return nil
	}
	if math.IsNaN(f.Float64) || math.IsInf(f.Float64, 0) {
		return nil
	}
	// Averages come back with a long decimal tail; two places is all the public
	// page and the dashboard ever show.
	rounded := math.Round(f.Float64*100) / 100
	return &rounded
}

// numericFromScore encodes an optional score for a NUMERIC column, with nil
// stored as NULL. It goes through the decimal text form rather than the
// Int/Exp fields directly, so 4.7 stores as 4.7 instead of a binary-float
// approximation of it.
func numericFromScore(f *float64) pgtype.Numeric {
	if f == nil || math.IsNaN(*f) || math.IsInf(*f, 0) {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	if err := n.Scan(strconv.FormatFloat(*f, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}
	}
	return n
}

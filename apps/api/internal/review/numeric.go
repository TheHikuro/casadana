package review

import (
	"math"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// numericToFloat reads a NUMERIC column as a score. A NULL, NaN or infinite
// value reads as 0: the display aggregates are "unset = not published", and a
// bogus stored value must not leak into the public JSON as NaN.
func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid || n.NaN {
		return 0
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	if math.IsNaN(f.Float64) || math.IsInf(f.Float64, 0) {
		return 0
	}
	return f.Float64
}

// numericFromFloat encodes a score for a NUMERIC column. It goes through the
// decimal text form rather than the Int/Exp fields directly, so 4.7 stores as
// 4.7 instead of a binary-float approximation of it. NaN and infinities become
// NULL — the column holds scores, and there is no sane score for them.
func numericFromFloat(f float64) pgtype.Numeric {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	if err := n.Scan(strconv.FormatFloat(f, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}
	}
	return n
}

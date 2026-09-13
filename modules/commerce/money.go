package commerce

import (
	"fmt"
	"strconv"
)

// ExactCents adapts an integer cent amount to legacy fmt templates while
// retaining exact decimal rendering for the floating-point verbs they use.
type ExactCents int64

func (value ExactCents) Format(state fmt.State, _ rune) {
	_, _ = state.Write([]byte(FormatCents(int64(value))))
}

// FormatCents renders a cent amount without floating-point conversion.
// Keeping this at the domain boundary prevents precision loss in provider
// payloads and user-facing financial values.
func FormatCents(cents int64) string {
	negative := cents < 0
	var value uint64
	if negative {
		value = uint64(-(cents + 1)) + 1 // safe for MinInt64
	} else {
		value = uint64(cents)
	}
	whole := value / 100
	minor := value % 100
	result := strconv.FormatUint(whole, 10) + "." + strconv.FormatUint(minor+100, 10)[1:]
	if negative {
		return "-" + result
	}
	return result
}

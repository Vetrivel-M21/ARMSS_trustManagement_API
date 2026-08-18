package shared

import (
	"trust-management/backend/internal/dto"

	"github.com/shopspring/decimal"
)

// SumDenominations computes the physical cash total from a denomination
// breakdown (value × quantity for each entry, summed). Negative quantities are
// ignored rather than subtracted, since a denomination count can never be
// negative in reality.
func SumDenominations(items []dto.DenominationItem) decimal.Decimal {
	sum := decimal.Zero
	for _, item := range items {
		if item.Quantity <= 0 {
			continue
		}
		sum = sum.Add(decimal.NewFromInt(int64(item.Value * item.Quantity)))
	}
	return sum
}

package shared

import (
	"testing"

	"trust-management/backend/internal/dto"

	"github.com/shopspring/decimal"
)

func TestSumDenominations(t *testing.T) {
	cases := []struct {
		name  string
		items []dto.DenominationItem
		want  string
	}{
		{
			name: "matches spec example: 2000x5 + 500x4 + 200x3",
			items: []dto.DenominationItem{
				{Value: 2000, Quantity: 5},
				{Value: 500, Quantity: 4},
				{Value: 200, Quantity: 3},
			},
			want: "12600",
		},
		{
			name: "single denomination",
			items: []dto.DenominationItem{
				{Value: 100, Quantity: 1},
			},
			want: "100",
		},
		{
			name:  "empty",
			items: []dto.DenominationItem{},
			want:  "0",
		},
		{
			name: "negative quantity ignored, not subtracted",
			items: []dto.DenominationItem{
				{Value: 500, Quantity: 2},
				{Value: 100, Quantity: -5},
			},
			want: "1000",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SumDenominations(tc.items)
			want, _ := decimal.NewFromString(tc.want)
			if !got.Equal(want) {
				t.Errorf("SumDenominations() = %s, want %s", got.String(), want.String())
			}
		})
	}
}

// TestSumDenominations_MismatchDetection mirrors the exact scenario in spec
// section 16: a ₹3,750 donation with denominations totaling ₹3,700 must be
// detected as a mismatch by the caller (donation.go/cash.go compare this sum
// against the claimed amount via decimal.Equal).
func TestSumDenominations_MismatchDetection(t *testing.T) {
	items := []dto.DenominationItem{
		{Value: 2000, Quantity: 1},
		{Value: 500, Quantity: 3},
		{Value: 200, Quantity: 1},
	}
	claimedAmount := decimal.NewFromInt(3750)

	sum := SumDenominations(items)
	if sum.Equal(claimedAmount) {
		t.Fatalf("expected denomination sum (%s) to NOT match claimed amount (%s)", sum, claimedAmount)
	}
	if sum.String() != "3700" {
		t.Fatalf("expected sum 3700, got %s", sum)
	}
}

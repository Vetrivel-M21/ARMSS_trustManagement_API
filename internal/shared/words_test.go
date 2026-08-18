package shared

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestConvertAmountToWords(t *testing.T) {
	cases := []struct {
		amount string
		want   string
	}{
		{"0", "Zero Rupees Only"},
		{"1", "Rupees One Only"},
		{"100", "Rupees One Hundred Only"},
		{"3750", "Rupees Three Thousand Seven Hundred Fifty Only"},
		{"5000", "Rupees Five Thousand Only"},
		{"100000", "Rupees One Lakh Only"},
		{"10000000", "Rupees One Crore Only"},
		{"5000.50", "Rupees Five Thousand and Fifty Paise Only"},
	}

	for _, tc := range cases {
		amt, err := decimal.NewFromString(tc.amount)
		if err != nil {
			t.Fatalf("failed to parse test amount %q: %v", tc.amount, err)
		}
		got := ConvertAmountToWords(amt)
		if got != tc.want {
			t.Errorf("ConvertAmountToWords(%s) = %q, want %q", tc.amount, got, tc.want)
		}
	}
}

func TestConvertAmountToWords_NegativeIsZero(t *testing.T) {
	got := ConvertAmountToWords(decimal.NewFromInt(-100))
	if got != "Zero Rupees Only" {
		t.Errorf("negative amount should render as Zero Rupees Only, got %q", got)
	}
}

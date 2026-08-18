package shared

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

var units = []string{"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen", "Seventeen", "Eighteen", "Nineteen"}
var tens = []string{"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety"}

func convertNumberToWords(n int64) string {
	if n == 0 {
		return ""
	}
	if n < 20 {
		return units[n]
	}
	if n < 100 {
		return tens[n/10] + " " + units[n%10]
	}
	if n < 1000 {
		return units[n/100] + " Hundred " + convertNumberToWords(n%100)
	}
	if n < 100000 { // Thousands (up to 99,999)
		return convertNumberToWords(n/1000) + " Thousand " + convertNumberToWords(n%1000)
	}
	if n < 10000000 { // Lakhs (up to 99,99,999)
		return convertNumberToWords(n/100000) + " Lakh " + convertNumberToWords(n%100000)
	}
	// Crores
	return convertNumberToWords(n/10000000) + " Crore " + convertNumberToWords(n%10000000)
}

func ConvertAmountToWords(amount decimal.Decimal) string {
	if amount.Sign() <= 0 {
		return "Zero Rupees Only"
	}

	rupees := amount.Truncate(0).IntPart()
	paise := amount.Sub(amount.Truncate(0)).Mul(decimal.NewFromInt(100)).Round(0).IntPart()

	rupeesWords := strings.TrimSpace(convertNumberToWords(rupees))
	if rupeesWords == "" {
		rupeesWords = "Zero"
	}

	result := fmt.Sprintf("Rupees %s Only", rupeesWords)
	if paise > 0 {
		paiseWords := strings.TrimSpace(convertNumberToWords(paise))
		result = fmt.Sprintf("Rupees %s and %s Paise Only", rupeesWords, paiseWords)
	}

	return result
}

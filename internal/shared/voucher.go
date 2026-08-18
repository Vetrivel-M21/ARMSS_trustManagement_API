package shared

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// VoucherCompanyCode identifies the trust in generated voucher numbers
// (distinct from the group's other companies, e.g. "AEMP" for the
// employment-solution arm).
const VoucherCompanyCode = "ACHT"

// GenerateVoucherNumber produces the next voucher number for the financial
// year containing bizDate, in the form "ACHT/{count}/{FY}" (e.g.
// "ACHT/1/26-27"). The counter resets to 1 at the start of each financial
// year since it's keyed per-FY and auto-seeds on first use.
func GenerateVoucherNumber(tx *gorm.DB, bizDate time.Time) (string, error) {
	fy := FinancialYearLabel(bizDate)
	seq, err := NextSequenceAutoSeed(tx, "VOUCHER_"+fy)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%d/%s", VoucherCompanyCode, seq, fy), nil
}

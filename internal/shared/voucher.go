package shared

import (
	"fmt"
	"strings"
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
	return GenerateBranchVoucherNumber(tx, "", bizDate)
}

// GenerateBranchVoucherNumber produces the next voucher number for a specific branch
// and financial year, in the form "ACHT-{BRANCH}/{count}/{FY}" or "ACHT/{count}/{FY}" for main branch.
func GenerateBranchVoucherNumber(tx *gorm.DB, branchCode string, bizDate time.Time) (string, error) {
	fy := FinancialYearLabel(bizDate)
	bCode := strings.ToUpper(strings.TrimSpace(branchCode))
	if bCode == "" || bCode == "MAIN" || bCode == "HQ" {
		bCode = VoucherCompanyCode
	} else {
		bCode = fmt.Sprintf("%s-%s", VoucherCompanyCode, bCode)
	}
	seqKey := fmt.Sprintf("VOUCHER_%s_%s", bCode, fy)
	seq, err := NextSequenceAutoSeed(tx, seqKey)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%d/%s", bCode, seq, fy), nil
}

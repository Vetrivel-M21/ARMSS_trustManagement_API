package shared

import (
	"database/sql"
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

	// Ensure sequence table is seeded to at least the max voucher number already in database
	var existingMax sql.NullInt64
	pattern := fmt.Sprintf("%s/%%/%s", bCode, fy)
	_ = tx.Raw(
		"SELECT MAX(CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(voucher_number, '/', 2), '/', -1) AS UNSIGNED)) FROM vouchers WHERE voucher_number LIKE ?",
		pattern,
	).Scan(&existingMax)

	if existingMax.Valid && existingMax.Int64 > 0 {
		_ = tx.Exec(
			"INSERT INTO sequences (name, current_value) VALUES (?, ?) ON DUPLICATE KEY UPDATE current_value = GREATEST(current_value, ?)",
			seqKey, existingMax.Int64, existingMax.Int64,
		)
	} else if bCode == VoucherCompanyCode {
		// Check legacy key VOUCHER_{fy}
		legacyKey := fmt.Sprintf("VOUCHER_%s", fy)
		var legacyVal sql.NullInt64
		_ = tx.Raw("SELECT current_value FROM sequences WHERE name = ?", legacyKey).Scan(&legacyVal)
		if legacyVal.Valid && legacyVal.Int64 > 0 {
			_ = tx.Exec(
				"INSERT INTO sequences (name, current_value) VALUES (?, ?) ON DUPLICATE KEY UPDATE current_value = GREATEST(current_value, ?)",
				seqKey, legacyVal.Int64, legacyVal.Int64,
			)
		}
	}

	for attempts := 0; attempts < 100; attempts++ {
		seq, err := NextSequenceAutoSeed(tx, seqKey)
		if err != nil {
			return "", err
		}
		candidate := fmt.Sprintf("%s/%d/%s", bCode, seq, fy)

		var count int64
		if err := tx.Table("vouchers").Where("voucher_number = ?", candidate).Count(&count).Error; err != nil {
			return candidate, nil
		}
		if count == 0 {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique voucher number for %s after multiple attempts", bCode)
}

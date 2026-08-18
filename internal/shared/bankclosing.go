package shared

import (
	"errors"

	"gorm.io/gorm"
)

// ErrBankAccountClosed is returned by EnsureBankAccountOpen when a bank
// account has already been closed (and not since unlocked) for a given
// business date.
var ErrBankAccountClosed = errors.New("bank account is closed for this business date")

// EnsureBankAccountOpen blocks new transactions from landing in a bank
// account/date that has already been closed (spec's day-closing lock,
// extended to the per-bank-account closing introduced alongside cash
// closing). A BankClosing row with status != 'UNLOCKED' means the account
// is locked for that date; correcting it requires an admin-approved
// BANK_DAY unlock request (see internal/unlock).
func EnsureBankAccountOpen(db *gorm.DB, bankAccountID uint, dateStr string) error {
	var count int64
	if err := db.Table("bank_closings").
		Where("bank_account_id = ? AND business_date = ? AND status != ?", bankAccountID, dateStr, "UNLOCKED").
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrBankAccountClosed
	}
	return nil
}

package shared

import (
	"gorm.io/gorm"

	"github.com/shopspring/decimal"
)

// CashFigures holds the recomputed cash-flow figures for a single business date.
type CashFigures struct {
	Inflow          decimal.Decimal
	Outflow         decimal.Decimal
	ExpectedClosing decimal.Decimal
}

// GetOpeningCash returns the physical cash count from the most recently CLOSED
// business day strictly before dateStr, so a new day's opening balance carries
// forward from the previous day's actual closing (standard cash-book behavior).
// Returns zero if no prior closed day exists (the very first business day).
func GetOpeningCash(db *gorm.DB, dateStr string) decimal.Decimal {
	var prevPhysical decimal.Decimal
	db.Table("daily_closings").
		Where("business_date < ? AND status = ?", dateStr, "CLOSED").
		Order("business_date DESC").
		Limit(1).
		Select("physical_cash_count").
		Scan(&prevPhysical)
	return prevPhysical
}

// GetOpeningBankBalance returns the actual closing balance from the most
// recently closed record strictly before dateStr for the given bank account.
// Falls back to the bank account's static OpeningBalance if this is the first
// closing ever recorded for that account.
func GetOpeningBankBalance(db *gorm.DB, bankAccountID uint, dateStr string, accountOpeningBalance decimal.Decimal) decimal.Decimal {
	var prevClosing decimal.Decimal
	err := db.Table("bank_closings").
		Where("bank_account_id = ? AND business_date < ?", bankAccountID, dateStr).
		Order("business_date DESC").
		Limit(1).
		Select("actual_closing").
		Scan(&prevClosing).Error
	if err != nil || prevClosing.IsZero() {
		var exists int64
		db.Table("bank_closings").
			Where("bank_account_id = ? AND business_date < ?", bankAccountID, dateStr).
			Count(&exists)
		if exists == 0 {
			return accountOpeningBalance
		}
	}
	return prevClosing
}

// RecomputeCashFigures sums cash_transactions for the given business date and
// derives the expected closing cash from the supplied opening balance. Callers
// must always call this rather than trusting a previously persisted
// ExpectedClosingCash/CashDifference value, which goes stale the moment a new
// cash transaction is recorded for that date.
func RecomputeCashFigures(db *gorm.DB, dateStr string, openingCash decimal.Decimal) CashFigures {
	var inflow decimal.Decimal
	db.Table("cash_transactions").
		Where("business_date = ? AND transaction_type = ?", dateStr, "INFLOW").
		Select("COALESCE(SUM(amount), 0)").Scan(&inflow)

	var outflow decimal.Decimal
	db.Table("cash_transactions").
		Where("business_date = ? AND transaction_type = ?", dateStr, "OUTFLOW").
		Select("COALESCE(SUM(amount), 0)").Scan(&outflow)

	return CashFigures{
		Inflow:          inflow,
		Outflow:         outflow,
		ExpectedClosing: openingCash.Add(inflow).Sub(outflow),
	}
}

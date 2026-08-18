package shared

import (
	"fmt"

	"gorm.io/gorm"
)

// NextSequence atomically increments and returns the next value for the named
// counter (e.g. "DONATION", "EXPENSE", "VOUCHER", "DONOR"). It must be called
// within an active transaction (tx) so the row-level lock taken by the UPDATE
// is held for the lifetime of the caller's transaction, serializing concurrent
// number generation and eliminating the last-ID-plus-one race condition.
func NextSequence(tx *gorm.DB, name string) (int64, error) {
	res := tx.Exec("UPDATE sequences SET current_value = current_value + 1 WHERE name = ?", name)
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, fmt.Errorf("sequence %q is not seeded", name)
	}

	var value int64
	if err := tx.Raw("SELECT current_value FROM sequences WHERE name = ?", name).Scan(&value).Error; err != nil {
		return 0, err
	}
	return value, nil
}

// NextSequenceAutoSeed behaves like NextSequence but creates the counter row
// (starting at 1) on first use instead of requiring it to be pre-seeded.
// Used for sequences whose key is only known at runtime — e.g. one counter
// per financial year — where pre-seeding every future key isn't possible.
func NextSequenceAutoSeed(tx *gorm.DB, name string) (int64, error) {
	if err := tx.Exec(
		"INSERT INTO sequences (name, current_value) VALUES (?, 1) ON DUPLICATE KEY UPDATE current_value = current_value + 1",
		name,
	).Error; err != nil {
		return 0, err
	}

	var value int64
	if err := tx.Raw("SELECT current_value FROM sequences WHERE name = ?", name).Scan(&value).Error; err != nil {
		return 0, err
	}
	return value, nil
}

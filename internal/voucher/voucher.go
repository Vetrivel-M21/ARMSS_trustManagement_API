package voucher

import (
	"net/http"
	"strconv"

	"trust-management/backend/internal/database"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

type VoucherHandler struct{}

func NewVoucherHandler() *VoucherHandler {
	return &VoucherHandler{}
}

func (h *VoucherHandler) GetVouchers(c *gin.Context) {
	var vouchers []models.Voucher
	query := database.DB.Order("id desc")

	if voucherType := c.Query("type"); voucherType != "" {
		query = query.Where("voucher_type = ?", voucherType)
	}

	if err := query.Find(&vouchers).Error; err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to fetch vouchers")
		return
	}

	shared.SendSuccess(c, http.StatusOK, vouchers)
}

// GetVoucherByID returns one voucher enriched with everything the receipt
// view needs in a single call: the donor's phone (for the WhatsApp
// Send-to-Vendor link), the linked scheme's food/meal type and category, the
// raw purpose/reference number as entered, and the relevant bank account (for
// its QR code) — or, for a CASH-mode voucher, the trust's designated default
// bank account (the earliest-created active account) so every receipt can
// still show a QR to pay/verify digitally.
func (h *VoucherHandler) GetVoucherByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "Invalid voucher ID")
		return
	}

	var voucher models.Voucher
	if err := database.DB.First(&voucher, id).Error; err != nil {
		shared.SendAppError(c, http.StatusNotFound, "Voucher not found")
		return
	}

	result := gin.H{
		"id":                  voucher.ID,
		"voucher_number":      voucher.VoucherNumber,
		"voucher_type":        voucher.VoucherType,
		"business_date":       voucher.BusinessDate,
		"source_type":         voucher.SourceType,
		"source_id":           voucher.SourceID,
		"payee_or_donor_name": voucher.PayeeOrDonorName,
		"amount":              voucher.Amount,
		"amount_in_words":     voucher.AmountInWords,
		"payment_mode":        voucher.PaymentMode,
		"status":              voucher.Status,
		"created_at":          voucher.CreatedAt,
	}

	var bankAccount *models.BankAccount

	if voucher.SourceType == "DONATION" {
		var donation models.Donation
		if err := database.DB.Preload("Donor").Preload("Scheme").First(&donation, voucher.SourceID).Error; err == nil {
			result["purpose"] = donation.Purpose
			result["reference_number"] = donation.ReferenceNumber
			if donation.Donor != nil {
				result["donor_phone"] = donation.Donor.Phone
				result["donor_father_name"] = donation.Donor.FatherName
			}
			if donation.Scheme != nil {
				result["food_type"] = donation.Scheme.FoodType
				result["meal_type"] = donation.Scheme.MealType
				result["category"] = donation.Scheme.Category
			}
			if donation.BankAccountID != nil {
				var acc models.BankAccount
				if err := database.DB.First(&acc, *donation.BankAccountID).Error; err == nil {
					bankAccount = &acc
				}
			}
		}
	} else if voucher.SourceType == "EXPENSE" {
		var expense models.Expense
		if err := database.DB.First(&expense, voucher.SourceID).Error; err == nil {
			result["purpose"] = expense.Description
			result["reference_number"] = expense.ReferenceNumber
			result["category"] = expense.Category
			if expense.BankAccountID != nil {
				var acc models.BankAccount
				if err := database.DB.First(&acc, *expense.BankAccountID).Error; err == nil {
					bankAccount = &acc
				}
			}
		}
	}

	if bankAccount == nil {
		var defaultAcc models.BankAccount
		// A cash voucher shows this account purely to give the payer a
		// scannable QR — prefer an active account that actually has one on
		// file over just the earliest-created account, which may not.
		err := database.DB.Where("is_active = ? AND qr_code_path IS NOT NULL AND qr_code_path <> ''", true).
			Order("id asc").First(&defaultAcc).Error
		if err != nil {
			err = database.DB.Where("is_active = ?", true).Order("id asc").First(&defaultAcc).Error
		}
		if err == nil {
			bankAccount = &defaultAcc
		}
	}
	result["bank_account"] = bankAccount

	shared.SendSuccess(c, http.StatusOK, result)
}

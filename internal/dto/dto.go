package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

// --- Bank Account DTOs ---
type CreateBankAccountRequest struct {
	BankName            string          `json:"bank_name" binding:"required"`
	AccountName         string          `json:"account_name" binding:"required"`
	AccountNumberMasked string          `json:"account_number_masked" binding:"required"`
	IFSCCode            string          `json:"ifsc_code" binding:"required"`
	Branch              string          `json:"branch" binding:"required"`
	Location            string          `json:"location"`
	OpeningBalance      decimal.Decimal `json:"opening_balance"`
	QRCodePath          string          `json:"qr_code_path"`
}

type UpdateBankAccountRequest struct {
	BankName            string `json:"bank_name"`
	AccountName         string `json:"account_name"`
	AccountNumberMasked string `json:"account_number_masked"`
	IFSCCode            string `json:"ifsc_code"`
	Branch              string `json:"branch"`
	Location            string `json:"location"`
	QRCodePath          string `json:"qr_code_path"`
	IsActive            *bool  `json:"is_active"`
}

// Amount validation for decimal.Decimal fields (Amount > 0) is performed manually
// in the handler — go-playground/validator's "gt"/"required" tags do not reliably
// evaluate struct types like decimal.Decimal.
type InterBankTransferRequest struct {
	FromAccountID   uint            `json:"from_account_id" binding:"required"`
	ToAccountID     uint            `json:"to_account_id" binding:"required"`
	Amount          decimal.Decimal `json:"amount"`
	ReferenceNumber string          `json:"reference_number"`
	Description     string          `json:"description"`
}

// --- Donor & Family DTOs ---
type CreateDonorFamilyMember struct {
	FullName     string `json:"full_name" binding:"required"`
	Relationship string `json:"relationship" binding:"required"`
	DateOfBirth  string `json:"date_of_birth" binding:"required"` // YYYY-MM-DD
	Notes        string `json:"notes"`
}

type CreateDonorRequest struct {
	FullName        string                    `json:"full_name" binding:"required"`
	FatherName      string                    `json:"father_name"`
	Phone           string                    `json:"phone" binding:"required"`
	Email           string                    `json:"email"`
	AddressLine     string                    `json:"address_line"`
	City            string                    `json:"city"`
	State           string                    `json:"state"`
	Pincode         string                    `json:"pincode"`
	DateOfBirth     string                    `json:"date_of_birth"`    // YYYY-MM-DD
	AnniversaryDate string                    `json:"anniversary_date"` // YYYY-MM-DD
	MaritalStatus   string                    `json:"marital_status"`   // SINGLE, MARRIED, etc.
	AadhaarNumber   string                    `json:"aadhaar_number"`
	AadhaarDocPath  string                    `json:"aadhaar_doc_path"`
	PANNumber       string                    `json:"pan_number"`
	PANDocPath      string                    `json:"pan_doc_path"`
	PhotoPath       string                    `json:"photo_path"`
	Notes           string                    `json:"notes"`
	FamilyMembers   []CreateDonorFamilyMember `json:"family_members"`
}

type UpdateDonorRequest struct {
	FullName        string                    `json:"full_name"`
	FatherName      string                    `json:"father_name"`
	Phone           string                    `json:"phone"`
	Email           string                    `json:"email"`
	AddressLine     string                    `json:"address_line"`
	City            string                    `json:"city"`
	State           string                    `json:"state"`
	Pincode         string                    `json:"pincode"`
	DateOfBirth     string                    `json:"date_of_birth"`
	AnniversaryDate string                    `json:"anniversary_date"`
	MaritalStatus   string                    `json:"marital_status"`
	AadhaarNumber   string                    `json:"aadhaar_number"`
	AadhaarDocPath  string                    `json:"aadhaar_doc_path"`
	PANNumber       string                    `json:"pan_number"`
	PANDocPath      string                    `json:"pan_doc_path"`
	PhotoPath       string                    `json:"photo_path"`
	Notes           string                    `json:"notes"`
	IsActive        *bool                     `json:"is_active"`
	FamilyMembers   []CreateDonorFamilyMember `json:"family_members"`
}

// --- Expense Category DTOs ---
type CreateExpenseCategoryRequest struct {
	Name string `json:"name" binding:"required"`
}

// --- Scheme DTOs ---
type CreateSchemeRequest struct {
	Name          string          `json:"name" binding:"required"`
	Category      string          `json:"category" binding:"required"`
	FoodType      string          `json:"food_type"` // VEG, NON_VEG, NA
	MealType      string          `json:"meal_type"` // BREAKFAST, LUNCH, DINNER, NA
	DefaultAmount decimal.Decimal `json:"default_amount"`
	Description   string          `json:"description"`
}

// CreateSchemeBulkCell is one Veg/Non-Veg x Breakfast/Lunch/Dinner price cell
// in the scheme matrix dialog. A cell with a zero/omitted DefaultAmount is
// skipped — not every combination needs to be created.
type CreateSchemeBulkCell struct {
	FoodType      string          `json:"food_type" binding:"required"` // VEG, NON_VEG
	MealType      string          `json:"meal_type" binding:"required"` // BREAKFAST, LUNCH, DINNER
	DefaultAmount decimal.Decimal `json:"default_amount"`
}

type CreateSchemeBulkRequest struct {
	Description string                 `json:"description"`
	Cells       []CreateSchemeBulkCell `json:"cells" binding:"required"`
}

// --- Bank Closing DTO ---
type SubmitBankClosingRequest struct {
	BusinessDate  string          `json:"business_date" binding:"required"`
	ActualClosing decimal.Decimal `json:"actual_closing"`
}

type BankClosingEntry struct {
	BankAccountID uint            `json:"bank_account_id" binding:"required"`
	ActualClosing decimal.Decimal `json:"actual_closing"`
}

// CloseAllBanksRequest closes every listed bank account for one business date
// in a single all-or-nothing action — if any account's statement amount
// doesn't match its expected closing, none of them are closed.
type CloseAllBanksRequest struct {
	BusinessDate string             `json:"business_date" binding:"required"`
	Closings     []BankClosingEntry `json:"closings" binding:"required"`
}

// --- Cash Denomination DTO ---
type DenominationItem struct {
	Value    int `json:"value" binding:"required"`    // 2000, 500, 200, 100, 50, 20, 10, 5, 2, 1
	Quantity int `json:"quantity" binding:"required"` // Count
}

// --- Donation DTOs ---
type CreateDonationRequest struct {
	DonorID             uint               `json:"donor_id" binding:"required"`
	BusinessDate        string             `json:"business_date"` // YYYY-MM-DD
	Amount              decimal.Decimal    `json:"amount"`
	PaymentMode         string             `json:"payment_mode" binding:"required"` // CASH / BANK
	Purpose             string             `json:"purpose" binding:"required"`
	SchemeID            *uint              `json:"scheme_id"`
	EventType           string             `json:"event_type"` // BIRTHDAY, ANNIVERSARY, CHILD_BIRTHDAY, MEMORIAL, OTHER
	EventPersonName     string             `json:"event_person_name"`
	EventDate           string             `json:"event_date"` // YYYY-MM-DD
	RelationshipToDonor string             `json:"relationship_to_donor"`
	FamilyMemberID      *uint              `json:"family_member_id"`
	BankAccountID       *uint              `json:"bank_account_id"`  // Required if PaymentMode == "BANK"
	ReferenceNumber     string             `json:"reference_number"` // Optional, BANK mode only
	AttachmentPath      string             `json:"attachment_path"`  // Optional, BANK mode only
	Notes               string             `json:"notes"`
	Denominations       []DenominationItem `json:"denominations"` // Optional for CASH mode verification
}

// --- Expense DTOs ---
type CreateExpenseRequest struct {
	BusinessDate    string          `json:"business_date"`                   // YYYY-MM-DD
	PaymentMode     string          `json:"payment_mode" binding:"required"` // CASH / BANK
	BankAccountID   *uint           `json:"bank_account_id"`                 // Required if PaymentMode == BANK
	Category        string          `json:"category" binding:"required"`
	Amount          decimal.Decimal `json:"amount"`
	PayeeName       string          `json:"payee_name" binding:"required"`
	Description     string          `json:"description"`
	ReferenceNumber string          `json:"reference_number"`
	AttachmentPath  string          `json:"attachment_path"`
}

// --- Cash Denomination Grid DTO ---
type SubmitDailyCashDenominationsRequest struct {
	BusinessDate  string             `json:"business_date" binding:"required"`
	Denominations []DenominationItem `json:"denominations" binding:"required"`
}

// --- Daily Closing DTOs ---
type SubmitDailyClosingRequest struct {
	BusinessDate string `json:"business_date" binding:"required"`
	Notes        string `json:"notes"`
}

// --- Unlock Request DTOs ---
type SubmitUnlockRequest struct {
	EntityType    string `json:"entity_type"`     // CASH_DAY (default) / BANK_DAY
	BankAccountID *uint  `json:"bank_account_id"` // required when EntityType == BANK_DAY
	BusinessDate  string `json:"business_date" binding:"required"`
	Reason        string `json:"reason" binding:"required"`
}

type ReviewUnlockRequest struct {
	Status      string `json:"status" binding:"required"` // APPROVED / REJECTED
	ReviewNotes string `json:"review_notes"`
}

// --- Report Response DTOs ---
type YoYMonthComparisonItem struct {
	MonthName       string          `json:"month_name"`
	CurrentYearAmt  decimal.Decimal `json:"current_year_amount"`
	PreviousYearAmt decimal.Decimal `json:"previous_year_amount"`
	VarianceAmount  decimal.Decimal `json:"variance_amount"`
	VariancePercent float64         `json:"variance_percent"`
}

type YoYMonthDonorItem struct {
	DonorID      uint            `json:"donor_id"`
	DonorName    string          `json:"donor_name"`
	DonorCode    string          `json:"donor_code"`
	Amount       decimal.Decimal `json:"amount"`
	BusinessDate string          `json:"business_date"`
	Purpose      string          `json:"purpose"`
	Year         int             `json:"year"`
}

type BirthdayItem struct {
	Type           string    `json:"type"` // DONOR / FAMILY_MEMBER / ANNIVERSARY
	DonorID        uint      `json:"donor_id"`
	DonorName      string    `json:"donor_name"`
	PersonName     string    `json:"person_name"`
	Phone          string    `json:"phone"`
	Email          string    `json:"email"`
	Relationship   string    `json:"relationship"`
	DateOfBirth    time.Time `json:"date_of_birth"`
	BirthdayDay    int       `json:"birthday_day"`
	BirthdayMonth  int       `json:"birthday_month"`
	Age            int       `json:"age"`
	FamilyMemberID *uint     `json:"family_member_id,omitempty"`
}

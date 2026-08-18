package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Role string

const (
	RoleStaff Role = "STAFF"
	RoleAdmin Role = "ADMIN"
)

type PaymentMode string

const (
	PaymentModeCash PaymentMode = "CASH"
	PaymentModeBank PaymentMode = "BANK"
)

type BusinessDayStatus string

const (
	DayStatusOpen         BusinessDayStatus = "OPEN"
	DayStatusReadyToClose BusinessDayStatus = "READY_TO_CLOSE"
	DayStatusClosed       BusinessDayStatus = "CLOSED"
	DayStatusUnlocked     BusinessDayStatus = "UNLOCKED"
)

type User struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"size:50;uniqueIndex;not null" json:"username"`
	FullName     string    `gorm:"size:100;not null" json:"full_name"`
	Email        string    `gorm:"size:100;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         Role      `gorm:"type:enum('STAFF','ADMIN');not null;default:'STAFF'" json:"role"`
	IsActive     bool      `gorm:"default:true;not null" json:"is_active"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type Donor struct {
	ID              uint                `gorm:"primaryKey;autoIncrement" json:"id"`
	DonorCode       string              `gorm:"size:20;uniqueIndex;not null" json:"donor_code"`
	FullName        string              `gorm:"size:150;not null" json:"full_name"`
	FatherName      string              `gorm:"size:150" json:"father_name"`
	Phone           string              `gorm:"size:20;index" json:"phone"`
	Email           string              `gorm:"size:100" json:"email"`
	AddressLine     string              `gorm:"type:text" json:"address_line"`
	City            string              `gorm:"size:50;index" json:"city"`
	State           string              `gorm:"size:50" json:"state"`
	Pincode         string              `gorm:"size:10" json:"pincode"`
	DateOfBirth     *time.Time          `gorm:"type:date;index" json:"date_of_birth"`
	AnniversaryDate *time.Time          `gorm:"type:date" json:"anniversary_date"`
	MaritalStatus   string              `gorm:"size:20" json:"marital_status"`
	AadhaarNumber   string              `gorm:"size:20" json:"aadhaar_number"`
	AadhaarDocPath  string              `gorm:"size:255" json:"aadhaar_doc_path"`
	PANNumber       string              `gorm:"size:20" json:"pan_number"`
	PANDocPath      string              `gorm:"size:255" json:"pan_doc_path"`
	PhotoPath       string              `gorm:"size:255" json:"photo_path"`
	Notes           string              `gorm:"type:text" json:"notes"`
	IsActive        bool                `gorm:"default:true;not null" json:"is_active"`
	CreatedAt       time.Time           `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time           `gorm:"autoUpdateTime" json:"updated_at"`
	FamilyMembers   []DonorFamilyMember `gorm:"foreignKey:DonorID" json:"family_members,omitempty"`
}

type DonorFamilyMember struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	DonorID      uint      `gorm:"not null;index" json:"donor_id"`
	FullName     string    `gorm:"size:150;not null" json:"full_name"`
	Relationship string    `gorm:"size:30;not null" json:"relationship"`
	DateOfBirth  time.Time `gorm:"type:date;not null;index" json:"date_of_birth"`
	Notes        string    `gorm:"size:255" json:"notes"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type Scheme struct {
	ID            uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string          `gorm:"size:100;not null" json:"name"`
	Category      string          `gorm:"size:50;not null" json:"category"`
	FoodType      string          `gorm:"size:20;default:'NA'" json:"food_type"`
	MealType      string          `gorm:"size:20;default:'NA'" json:"meal_type"`
	DefaultAmount decimal.Decimal `gorm:"type:decimal(15,2);not null;default:0.00" json:"default_amount"`
	Description   string          `gorm:"type:text" json:"description"`
	IsActive      bool            `gorm:"default:true;not null" json:"is_active"`
	CreatedAt     time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

// ExpenseCategory is an admin-managed lookup list for Expense.Category —
// deactivating one (instead of deleting) preserves the category text on any
// expense already recorded against it.
type ExpenseCategory struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"size:100;not null;uniqueIndex" json:"name"`
	IsActive  bool      `gorm:"default:true;not null" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type Donation struct {
	ID                  uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	DonationNumber      string          `gorm:"size:30;uniqueIndex;not null" json:"donation_number"`
	DonorID             uint            `gorm:"not null;index" json:"donor_id"`
	Donor               *Donor          `gorm:"foreignKey:DonorID" json:"donor,omitempty"`
	BusinessDate        time.Time       `gorm:"type:date;not null;index" json:"business_date"`
	Amount              decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	PaymentMode         PaymentMode     `gorm:"type:enum('CASH','BANK');not null" json:"payment_mode"`
	Purpose             string          `gorm:"size:100;not null" json:"purpose"`
	SchemeID            *uint           `gorm:"index" json:"scheme_id"`
	Scheme              *Scheme         `gorm:"foreignKey:SchemeID" json:"scheme,omitempty"`
	EventType           string          `gorm:"size:100" json:"event_type"`
	EventPersonName     string          `gorm:"size:150" json:"event_person_name"`
	EventDate           *time.Time      `gorm:"type:date" json:"event_date"`
	RelationshipToDonor string          `gorm:"size:30" json:"relationship_to_donor"`
	FamilyMemberID      *uint           `gorm:"index" json:"family_member_id"`
	BankAccountID       *uint           `gorm:"index" json:"bank_account_id"`
	ReferenceNumber     string          `gorm:"size:100" json:"reference_number"`
	AttachmentPath      string          `gorm:"size:255" json:"attachment_path"`
	Notes               string          `gorm:"type:text" json:"notes"`
	Status              string          `gorm:"size:20;not null;default:'ACTIVE'" json:"status"`
	CreatedByID         uint            `gorm:"not null" json:"created_by"`
	CreatedBy           *User           `gorm:"foreignKey:CreatedByID" json:"created_by_user,omitempty"`
	CreatedAt           time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

type CashTransaction struct {
	ID              uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	BusinessDate    time.Time       `gorm:"type:date;not null;index" json:"business_date"`
	TransactionType string          `gorm:"size:20;not null" json:"transaction_type"` // INFLOW / OUTFLOW
	Amount          decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	SourceType      string          `gorm:"size:30;not null" json:"source_type"` // DONATION, EXPENSE, OTHER
	SourceID        uint            `gorm:"not null" json:"source_id"`
	Description     string          `gorm:"size:255;not null" json:"description"`
	CreatedByID     uint            `gorm:"not null" json:"created_by"`
	CreatedAt       time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

type CashDenomination struct {
	ID                uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	EntityType        string          `gorm:"size:30;not null;index" json:"entity_type"` // DONATION / DAILY_CLOSING
	EntityID          uint            `gorm:"not null;index" json:"entity_id"`
	DenominationValue int             `gorm:"not null" json:"denomination_value"`
	Quantity          int             `gorm:"not null;default:0" json:"quantity"`
	TotalAmount       decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"total_amount"`
}

type BankAccount struct {
	ID                  uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	BankName            string          `gorm:"size:100;not null" json:"bank_name"`
	AccountName         string          `gorm:"size:100;not null" json:"account_name"`
	AccountNumberMasked string          `gorm:"size:30;not null" json:"account_number_masked"`
	IFSCCode            string          `gorm:"size:20;not null" json:"ifsc_code"`
	Branch              string          `gorm:"size:100;not null" json:"branch"`
	OpeningBalance      decimal.Decimal `gorm:"type:decimal(15,2);not null;default:0.00" json:"opening_balance"`
	QRCodePath          string          `gorm:"size:255" json:"qr_code_path"`
	CurrentBalance      decimal.Decimal `gorm:"type:decimal(15,2);not null;default:0.00" json:"current_balance"`
	IsActive            bool            `gorm:"default:true;not null" json:"is_active"`
	CreatedAt           time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

type BankTransaction struct {
	ID              uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	BankAccountID   uint            `gorm:"not null;index" json:"bank_account_id"`
	BankAccount     *BankAccount    `gorm:"foreignKey:BankAccountID" json:"bank_account,omitempty"`
	BusinessDate    time.Time       `gorm:"type:date;not null;index" json:"business_date"`
	TransactionType string          `gorm:"size:20;not null" json:"transaction_type"` // CREDIT / DEBIT
	Amount          decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	Category        string          `gorm:"size:50;not null" json:"category"`
	ReferenceNumber string          `gorm:"size:100;index" json:"reference_number"`
	SourceType      string          `gorm:"size:30;not null" json:"source_type"`
	SourceID        uint            `gorm:"not null" json:"source_id"`
	Description     string          `gorm:"type:text" json:"description"`
	CreatedByID     uint            `gorm:"not null" json:"created_by"`
	CreatedAt       time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

type Expense struct {
	ID              uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	ExpenseNumber   string          `gorm:"size:30;uniqueIndex;not null" json:"expense_number"`
	BusinessDate    time.Time       `gorm:"type:date;not null;index" json:"business_date"`
	PaymentMode     PaymentMode     `gorm:"type:enum('CASH','BANK');not null" json:"payment_mode"`
	BankAccountID   *uint           `gorm:"index" json:"bank_account_id"`
	BankAccount     *BankAccount    `gorm:"foreignKey:BankAccountID" json:"bank_account,omitempty"`
	Category        string          `gorm:"size:100;not null" json:"category"`
	Amount          decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	PayeeName       string          `gorm:"size:150;not null" json:"payee_name"`
	Description     string          `gorm:"type:text" json:"description"`
	ReferenceNumber string          `gorm:"size:100" json:"reference_number"`
	AttachmentPath  string          `gorm:"size:255" json:"attachment_path"`
	Status          string          `gorm:"size:20;not null;default:'ACTIVE'" json:"status"`
	CreatedByID     uint            `gorm:"not null" json:"created_by"`
	CreatedAt       time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

type DailyClosing struct {
	ID                  uint              `gorm:"primaryKey;autoIncrement" json:"id"`
	BusinessDate        time.Time         `gorm:"type:date;uniqueIndex;not null" json:"business_date"`
	Status              BusinessDayStatus `gorm:"type:enum('OPEN','READY_TO_CLOSE','CLOSED','UNLOCKED');not null;default:'OPEN'" json:"status"`
	OpeningCash         decimal.Decimal   `gorm:"type:decimal(15,2);not null" json:"opening_cash"`
	CashInflow          decimal.Decimal   `gorm:"type:decimal(15,2);not null;default:0.00" json:"cash_inflow"`
	CashOutflow         decimal.Decimal   `gorm:"type:decimal(15,2);not null;default:0.00" json:"cash_outflow"`
	ExpectedClosingCash decimal.Decimal   `gorm:"type:decimal(15,2);not null" json:"expected_closing_cash"`
	PhysicalCashCount   decimal.Decimal   `gorm:"type:decimal(15,2);not null;default:0.00" json:"physical_cash_count"`
	CashDifference      decimal.Decimal   `gorm:"type:decimal(15,2);not null;default:0.00" json:"cash_difference"`
	ClosedByID          *uint             `json:"closed_by"`
	ClosedBy            *User             `gorm:"foreignKey:ClosedByID" json:"closed_by_user,omitempty"`
	ClosedAt            *time.Time        `json:"closed_at"`
}

type BankClosing struct {
	ID              uint              `gorm:"primaryKey;autoIncrement" json:"id"`
	BusinessDate    time.Time         `gorm:"type:date;not null;index" json:"business_date"`
	BankAccountID   uint              `gorm:"not null;index" json:"bank_account_id"`
	BankAccount     *BankAccount      `gorm:"foreignKey:BankAccountID" json:"bank_account,omitempty"`
	OpeningBalance  decimal.Decimal   `gorm:"type:decimal(15,2);not null" json:"opening_balance"`
	TotalCredits    decimal.Decimal   `gorm:"type:decimal(15,2);not null;default:0.00" json:"total_credits"`
	TotalDebits     decimal.Decimal   `gorm:"type:decimal(15,2);not null;default:0.00" json:"total_debits"`
	ExpectedClosing decimal.Decimal   `gorm:"type:decimal(15,2);not null" json:"expected_closing"`
	ActualClosing   decimal.Decimal   `gorm:"type:decimal(15,2);not null" json:"actual_closing"`
	Difference      decimal.Decimal   `gorm:"type:decimal(15,2);not null;default:0.00" json:"difference"`
	Status          BusinessDayStatus `gorm:"type:varchar(20);not null;default:'CLOSED'" json:"status"`
}

type Voucher struct {
	ID               uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	VoucherNumber    string          `gorm:"size:30;uniqueIndex;not null" json:"voucher_number"`
	VoucherType      string          `gorm:"size:30;not null" json:"voucher_type"` // DONATION_RECEIPT / EXPENSE_VOUCHER
	BusinessDate     time.Time       `gorm:"type:date;not null" json:"business_date"`
	SourceType       string          `gorm:"size:30;not null" json:"source_type"` // DONATION / EXPENSE
	SourceID         uint            `gorm:"not null" json:"source_id"`
	PayeeOrDonorName string          `gorm:"size:150;not null" json:"payee_or_donor_name"`
	Amount           decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	AmountInWords    string          `gorm:"type:text;not null" json:"amount_in_words"`
	PaymentMode      PaymentMode     `gorm:"type:enum('CASH','BANK');not null" json:"payment_mode"`
	Status           string          `gorm:"size:20;not null;default:'ISSUED'" json:"status"`
	CreatedByID      uint            `gorm:"not null" json:"created_by"`
	CreatedAt        time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

type UnlockRequest struct {
	ID            uint         `gorm:"primaryKey;autoIncrement" json:"id"`
	EntityType    string       `gorm:"size:20;not null;default:'CASH_DAY'" json:"entity_type"` // CASH_DAY, BANK_DAY
	BankAccountID *uint        `gorm:"index" json:"bank_account_id"`
	BankAccount   *BankAccount `gorm:"foreignKey:BankAccountID" json:"bank_account,omitempty"`
	BusinessDate  time.Time    `gorm:"type:date;not null;index" json:"business_date"`
	RequestedByID uint         `gorm:"not null" json:"requested_by"`
	RequestedBy   *User        `gorm:"foreignKey:RequestedByID" json:"requested_by_user,omitempty"`
	RequestReason string       `gorm:"type:text;not null" json:"request_reason"`
	Status        string       `gorm:"size:20;not null;default:'PENDING'" json:"status"` // PENDING, APPROVED, REJECTED
	ReviewedByID  *uint        `json:"reviewed_by"`
	ReviewedBy    *User        `gorm:"foreignKey:ReviewedByID" json:"reviewed_by_user,omitempty"`
	ReviewReason  string       `gorm:"type:text" json:"review_reason"`
	RequestedAt   time.Time    `gorm:"autoCreateTime" json:"requested_at"`
	ReviewedAt    *time.Time   `json:"reviewed_at"`
}

type AuditLog struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     *uint     `gorm:"index" json:"user_id"`
	User       *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Action     string    `gorm:"size:100;not null;index" json:"action"`
	EntityName string    `gorm:"size:50;not null;index" json:"entity_name"`
	EntityID   uint      `gorm:"not null;index" json:"entity_id"`
	BeforeData string    `gorm:"type:json" json:"before_data"`
	AfterData  string    `gorm:"type:json" json:"after_data"`
	Reason     string    `gorm:"type:text" json:"reason"`
	IPAddress  string    `gorm:"size:45" json:"ip_address"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// Sequence provides atomic, concurrency-safe number generation for donation/expense/
// voucher/donor codes. Rows are seeded per key and incremented under row-level locking
// inside the caller's transaction (see internal/shared.NextSequence).
type Sequence struct {
	Name         string `gorm:"primaryKey;size:30" json:"name"`
	CurrentValue int64  `gorm:"not null;default:0" json:"current_value"`
}

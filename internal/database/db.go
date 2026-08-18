package database

import (
	"fmt"
	"time"

	"trust-management/backend/internal/config"
	"trust-management/backend/internal/models"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.GetDSN()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL database: %w", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	log.Info().Msg("Successfully connected to MySQL database")

	// Run versioned schema migrations (backend/migrations/*.sql) — the authoritative
	// schema-management mechanism. See internal/database/migrate.go.
	if err := RunVersionedMigrations(cfg); err != nil {
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	// Seed atomic numbering sequences (donor/donation/expense/voucher counters)
	if err := seedSequences(db); err != nil {
		return nil, fmt.Errorf("failed to seed numbering sequences: %w", err)
	}

	// Seed Development Data
	if err := Seed(db); err != nil {
		log.Warn().Err(err).Msg("Database seeding completed with warnings")
	}

	DB = db
	return db, nil
}

// seedSequences ensures the atomic-numbering counters exist. On first creation each
// counter is seeded from the highest existing row ID in its corresponding table, so
// upgrading a database that already has donations/expenses/vouchers/donors does not
// generate colliding numbers. It never resets an existing counter, so it is safe to
// call on every startup.
func seedSequences(db *gorm.DB) error {
	seedFrom := map[string]interface{}{
		"DONOR":    &models.Donor{},
		"DONATION": &models.Donation{},
		"EXPENSE":  &models.Expense{},
		"VOUCHER":  &models.Voucher{},
	}
	for name, table := range seedFrom {
		var existing models.Sequence
		if err := db.Where("name = ?", name).First(&existing).Error; err == nil {
			continue // already seeded; never overwrite
		}
		var maxID int64
		if err := db.Model(table).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error; err != nil {
			return err
		}
		seq := models.Sequence{Name: name, CurrentValue: maxID}
		if err := db.Create(&seq).Error; err != nil {
			return err
		}
	}
	return nil
}

func Seed(db *gorm.DB) error {
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return nil // Already seeded
	}

	log.Info().Msg("Seeding initial development users and master data...")

	// Hash password
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("Admin@123"), bcrypt.DefaultCost)
	staffHash, _ := bcrypt.GenerateFromPassword([]byte("Staff@123"), bcrypt.DefaultCost)

	adminUser := models.User{
		Username:     "admin",
		FullName:     "System Administrator",
		Email:        "admin@trust.org",
		PasswordHash: string(adminHash),
		Role:         models.RoleAdmin,
		IsActive:     true,
	}

	staffUser := models.User{
		Username:     "staff",
		FullName:     "Office Staff",
		Email:        "staff@trust.org",
		PasswordHash: string(staffHash),
		Role:         models.RoleStaff,
		IsActive:     true,
	}

	if err := db.Create(&adminUser).Error; err != nil {
		return err
	}
	if err := db.Create(&staffUser).Error; err != nil {
		return err
	}

	// Seed Sample Schemes
	schemes := []models.Scheme{
		{
			Name:          "Annadhanam Lunch (Veg)",
			Category:      "FOOD",
			FoodType:      "VEG",
			MealType:      "LUNCH",
			DefaultAmount: decimal.NewFromFloat(5000.00),
			Description:   "Full afternoon vegetarian lunch sponsorship for 100 people",
			IsActive:      true,
		},
		{
			Name:          "Annadhanam Lunch (Non-Veg)",
			Category:      "FOOD",
			FoodType:      "NON_VEG",
			MealType:      "LUNCH",
			DefaultAmount: decimal.NewFromFloat(7500.00),
			Description:   "Full afternoon non-vegetarian meal sponsorship",
			IsActive:      true,
		},
		{
			Name:          "Breakfast Special",
			Category:      "FOOD",
			FoodType:      "VEG",
			MealType:      "BREAKFAST",
			DefaultAmount: decimal.NewFromFloat(3000.00),
			Description:   "Morning tiffin and tea distribution",
			IsActive:      true,
		},
		{
			Name:          "Educational Scholarship Fund",
			Category:      "EDUCATION",
			FoodType:      "NA",
			MealType:      "NA",
			DefaultAmount: decimal.NewFromFloat(10000.00),
			Description:   "Financial aid for students in need",
			IsActive:      true,
		},
		{
			Name:          "Medical Care Relief",
			Category:      "MEDICAL",
			FoodType:      "NA",
			MealType:      "NA",
			DefaultAmount: decimal.NewFromFloat(5000.00),
			Description:   "Emergency healthcare support fund",
			IsActive:      true,
		},
	}
	for _, s := range schemes {
		db.Create(&s)
	}

	// Seed Sample Bank Accounts
	bankAccounts := []models.BankAccount{
		{
			BankName:            "State Bank of India (SBI)",
			AccountName:         "Trust Primary Operating Account",
			AccountNumberMasked: "**** **** 4892",
			IFSCCode:            "SBIN0001234",
			Branch:              "Main Branch",
			OpeningBalance:      decimal.NewFromFloat(250000.00),
			CurrentBalance:      decimal.NewFromFloat(250000.00),
			IsActive:            true,
		},
		{
			BankName:            "HDFC Bank",
			AccountName:         "Trust Donation Collection Account",
			AccountNumberMasked: "**** **** 9102",
			IFSCCode:            "HDFC0005678",
			Branch:              "Central Market Branch",
			OpeningBalance:      decimal.NewFromFloat(150000.00),
			CurrentBalance:      decimal.NewFromFloat(150000.00),
			IsActive:            true,
		},
	}
	for _, b := range bankAccounts {
		db.Create(&b)
	}

	log.Info().Msg("Database seeding completed successfully.")
	return nil
}

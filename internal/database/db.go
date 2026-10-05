package database

import (
	"fmt"
	"time"

	"trust-management/backend/internal/config"
	"trust-management/backend/internal/models"

	"github.com/rs/zerolog/log"
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

	// Ensure the 3 operational branches (Old Age Home, Children Home, Children Adoption Home) exist
	if err := EnsureDefaultBranches(db); err != nil {
		log.Warn().Err(err).Msg("Failed to verify default branches")
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

	log.Info().Msg("Seeding initial development users...")

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

	log.Info().Msg("Database seeding completed successfully.")
	return nil
}

// EnsureDefaultBranches guarantees that the three primary operational branches exist:
// 1: Old Age Home (OAH) / முதியோர்கள் இல்லம்
// 2: Children Home (CH) / குழந்தைகள் இல்லம்
// 3: Children Adoption Home (CAH) / சிறப்பு தத்தெடுத்தல் மையம்
func EnsureDefaultBranches(db *gorm.DB) error {
	// 1. Check Branch 1: If MAIN or empty, update to Old Age Home
	var b1 models.Branch
	if err := db.First(&b1, 1).Error; err == nil {
		if b1.BranchCode == "MAIN" || b1.Name == "Head Office / Main Branch" || b1.BranchCode == "" {
			db.Model(&b1).Updates(map[string]interface{}{
				"branch_code":    "OAH",
				"name":           "Old Age Home",
				"tamil_name":     "முதியோர்கள் இல்லம்",
				"license_number": "DSD/MDU/OAH/2024",
				"incharge_name":  "Dr. K. Murugan",
				"city":           "Madurai",
				"state":          "Tamil Nadu",
				"is_active":      true,
			})
		}
	} else {
		b1 = models.Branch{
			ID:            1,
			BranchCode:    "OAH",
			Name:          "Old Age Home",
			TamilName:     "முதியோர்கள் இல்லம்",
			LicenseNumber: "DSD/MDU/OAH/2024",
			InchargeName:  "Dr. K. Murugan",
			City:          "Madurai",
			State:         "Tamil Nadu",
			IsActive:      true,
		}
		_ = db.Create(&b1)
	}

	// 2. Branch 2: Children Home
	var b2 models.Branch
	if err := db.First(&b2, 2).Error; err != nil {
		b2 = models.Branch{
			ID:            2,
			BranchCode:    "CH",
			Name:          "Children Home",
			TamilName:     "குழந்தைகள் இல்லம்",
			LicenseNumber: "DSD/MDU/CH/2024",
			InchargeName:  "Mrs. S. Meenakshi",
			City:          "Madurai",
			State:         "Tamil Nadu",
			IsActive:      true,
		}
		_ = db.Create(&b2)
	}

	// 3. Branch 3: Children Adoption Home
	var b3 models.Branch
	if err := db.First(&b3, 3).Error; err != nil {
		b3 = models.Branch{
			ID:            3,
			BranchCode:    "CAH",
			Name:          "Children Adoption Home",
			TamilName:     "சிறப்பு தத்தெடுத்தல் மையம்",
			LicenseNumber: "DSD/MDU/SAA/2024",
			InchargeName:  "Dr. R. Anitha",
			City:          "Madurai",
			State:         "Tamil Nadu",
			IsActive:      true,
		}
		_ = db.Create(&b3)
	}

	// 4. Sync UPI and QR code paths from trust_home_configs into branches if present
	var homes []models.TrustHomeConfig
	if err := db.Find(&homes).Error; err == nil {
		for _, h := range homes {
			switch h.HomeKey {
			case "OLD_AGE_HOME":
				if h.UPIID != "" || h.QRCodePath != "" {
					db.Model(&models.Branch{}).Where("id = ? OR branch_code = ?", 1, "OAH").Updates(map[string]interface{}{
						"upi_id":       h.UPIID,
						"qr_code_path": h.QRCodePath,
					})
				}
			case "CHILDREN_HOME":
				if h.UPIID != "" || h.QRCodePath != "" {
					db.Model(&models.Branch{}).Where("id = ? OR branch_code = ?", 2, "CH").Updates(map[string]interface{}{
						"upi_id":       h.UPIID,
						"qr_code_path": h.QRCodePath,
					})
				}
			case "ADOPTION_HOME":
				if h.UPIID != "" || h.QRCodePath != "" {
					db.Model(&models.Branch{}).Where("id = ? OR branch_code = ?", 3, "CAH").Updates(map[string]interface{}{
						"upi_id":       h.UPIID,
						"qr_code_path": h.QRCodePath,
					})
				}
			}
		}
	}

	return nil
}


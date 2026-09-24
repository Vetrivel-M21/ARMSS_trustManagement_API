package main

import (
	"fmt"
	"os"

	"trust-management/backend/internal/auth"
	"trust-management/backend/internal/bank"
	"trust-management/backend/internal/cash"
	"trust-management/backend/internal/closing"
	"trust-management/backend/internal/config"
	"trust-management/backend/internal/database"
	"trust-management/backend/internal/donor"
	"trust-management/backend/internal/donation"
	"trust-management/backend/internal/expense"
	"trust-management/backend/internal/expensecategory"
	"trust-management/backend/internal/ledger"
	"trust-management/backend/internal/middleware"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/report"
	"trust-management/backend/internal/scheme"
	"trust-management/backend/internal/shared"
	"trust-management/backend/internal/unlock"
	"trust-management/backend/internal/upload"
	"trust-management/backend/internal/users"
	"trust-management/backend/internal/voucher"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

func main() {
	// 1. Initialize Logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Marshal decimal.Decimal as a bare JSON number (not a quoted string) so
	// existing frontend code that expects numeric amount fields keeps working.
	decimal.MarshalJSONWithoutQuotes = true

	log.Info().Msg("Starting Trust Management & Donation Accounting Backend Server...")

	// 2. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// 3. Connect to Database & Run Migrations
	_, err = database.InitDB(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Database initialization failed")
	}

	// 4. Setup Gin Engine
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()

	// 5. Global Middlewares
	r.Use(gin.Recovery())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.CORSMiddleware(cfg.FrontendURL, cfg.CORSAllowedOrigins))
	// Donor/bank KYC documents (Aadhaar, PAN, QR codes) are sensitive — require
	// a valid session to fetch them, not just an unguessable filename.
	uploadsGroup := r.Group("/uploads", middleware.AuthMiddleware(cfg.JWTSecret))
	uploadsGroup.Static("/", "./uploads")

	// 6. Register API Routes
	api := r.Group("/api/v1")
	{
		// Health Check
		api.GET("/health", func(c *gin.Context) {
			shared.SendSuccess(c, 200, gin.H{
				"status":   "healthy",
				"system":   "Trust Management API Backend",
				"version":  "1.0.0",
				"date_ist": shared.GetCurrentBusinessDate().Format("2006-01-02"),
			})
		})

		// Auth Module
		authHandler := auth.NewAuthHandler(cfg)
		authRoutes := api.Group("/auth")
		{
			authRoutes.POST("/login", authHandler.Login)
			authRoutes.POST("/logout", authHandler.Logout)
			authRoutes.GET("/me", middleware.AuthMiddleware(cfg.JWTSecret), authHandler.Me)
		}

		// Public Donor Mobile App Routes
		publicDonationHandler := donation.NewPublicDonationHandler(cfg)
		bankHandlerPublic := bank.NewBankHandler(cfg)
		schemeHandlerPublic := scheme.NewSchemeHandler()
		api.GET("/public/app-donation-config", bankHandlerPublic.GetAppDonationConfig)
		api.GET("/public/schemes/active", schemeHandlerPublic.GetPublicActiveSchemes)
		api.POST("/public/donations/create-order", publicDonationHandler.CreateRazorpayOrder)
		api.POST("/public/donations", publicDonationHandler.CreatePublicDonation)
		api.GET("/public/donations/history", publicDonationHandler.GetPublicDonorHistory)
		api.GET("/public/dashboard", publicDonationHandler.GetPublicDashboard)

		// Authenticated Routes Group — standard user JWT check
		protected := api.Group("", middleware.AuthMiddleware(cfg.JWTSecret))
		closedDayCheck := middleware.ClosedDayProtectionMiddleware()
		{
			// Bank Accounts
			bankHandler := bank.NewBankHandler()
			protected.GET("/bank-accounts", bankHandler.GetBankAccounts)
			protected.GET("/bank-accounts/active", bankHandler.GetActiveBankAccounts)
			protected.POST("/bank-accounts", middleware.RequireRole(models.RoleAdmin), bankHandler.CreateBankAccount)
			protected.PUT("/bank-accounts/:id", middleware.RequireRole(models.RoleAdmin), bankHandler.UpdateBankAccount)
			protected.PUT("/bank-accounts/:id/set-app-donation", middleware.RequireRole(models.RoleAdmin), bankHandler.SetAppDonationAccount)
			protected.GET("/bank-accounts/:id/transactions", bankHandler.GetBankTransactions)
			protected.GET("/bank-accounts/day-summary", bankHandler.GetBankDaySummary)
			protected.GET("/bank-accounts/:id/closing-status", bankHandler.GetBankClosingStatus)
			protected.POST("/bank-accounts/:id/close", closedDayCheck, bankHandler.CloseBankDay)
			protected.POST("/bank-accounts/close-all", closedDayCheck, bankHandler.CloseAllBankDays)
			protected.POST("/bank-accounts/transfer", closedDayCheck, bankHandler.TransferFunds)

			// User Management (Admin only)
			userHandler := users.NewUserHandler()
			protected.GET("/users", middleware.RequireRole(models.RoleAdmin), userHandler.GetUsers)
			protected.POST("/users", middleware.RequireRole(models.RoleAdmin), userHandler.CreateUser)
			protected.PUT("/users/:id", middleware.RequireRole(models.RoleAdmin), userHandler.UpdateUser)

			// Donors
			donorHandler := donor.NewDonorHandler()
			protected.GET("/donors", donorHandler.GetDonors)
			protected.GET("/donors/:id", donorHandler.GetDonorByID)
			protected.POST("/donors", donorHandler.CreateDonor)
			protected.PUT("/donors/:id", donorHandler.UpdateDonor)

			// Schemes
			schemeHandler := scheme.NewSchemeHandler()
			protected.GET("/schemes", schemeHandler.GetSchemes)
			protected.GET("/schemes/active", schemeHandler.GetActiveSchemes)
			protected.POST("/schemes", schemeHandler.CreateScheme)
			protected.POST("/schemes/bulk", schemeHandler.CreateSchemesBulk)
			protected.PUT("/schemes/:id", schemeHandler.UpdateScheme)

			// Donations
			donationHandler := donation.NewDonationHandler()
			protected.GET("/donations", donationHandler.GetDonations)
			protected.GET("/donations/:id", donationHandler.GetDonationByID)
			protected.POST("/donations", closedDayCheck, donationHandler.CreateDonation)

			// Cash Management
			cashHandler := cash.NewCashHandler()
			protected.GET("/cash/summary", cashHandler.GetDailyCashSummary)
			protected.POST("/cash/denominations", closedDayCheck, cashHandler.SubmitCashDenominations)

			// Expenses
			expenseHandler := expense.NewExpenseHandler()
			protected.GET("/expenses", expenseHandler.GetExpenses)
			protected.POST("/expenses", closedDayCheck, expenseHandler.CreateExpense)
			protected.POST("/expenses/:id/approve", middleware.RequireRole(models.RoleAdmin), expenseHandler.ApproveExpense)
			protected.POST("/expenses/:id/reject", middleware.RequireRole(models.RoleAdmin), expenseHandler.RejectExpense)

			// Expense Categories — list is available to all staff (needed by the
			// expense-creation form), create/update are Admin-only.
			expenseCategoryHandler := expensecategory.NewExpenseCategoryHandler()
			protected.GET("/expense-categories", expenseCategoryHandler.GetExpenseCategories)
			protected.GET("/expense-categories/active", expenseCategoryHandler.GetActiveExpenseCategories)
			protected.POST("/expense-categories", middleware.RequireRole(models.RoleAdmin), expenseCategoryHandler.CreateExpenseCategory)
			protected.PUT("/expense-categories/:id", middleware.RequireRole(models.RoleAdmin), expenseCategoryHandler.UpdateExpenseCategory)

			// Daily Closing State Machine
			closingHandler := closing.NewClosingHandler()
			protected.GET("/closing/status", closingHandler.GetDailyClosingStatus)
			// ExecuteDailyClosing has its own explicit "already CLOSED" check with a
			// clearer message — the generic closed-day middleware isn't needed here.
			protected.POST("/closing/execute", closingHandler.ExecuteDailyClosing)

			// Unlock & Audit
			unlockHandler := unlock.NewUnlockHandler()
			protected.GET("/unlock-requests", unlockHandler.GetUnlockRequests)
			protected.POST("/unlock-requests", unlockHandler.SubmitUnlockRequest)
			protected.PUT("/unlock-requests/:id/review", middleware.RequireRole(models.RoleAdmin), unlockHandler.ReviewUnlockRequest)
			protected.GET("/audit-logs", middleware.RequireRole(models.RoleAdmin), unlockHandler.GetAuditLogs)

			// Ledgers & Titles (Chart of Accounts)
			ledgerHandler := ledger.NewLedgerHandler()
			protected.GET("/ledgers", ledgerHandler.GetLedgers)
			protected.POST("/ledgers", ledgerHandler.CreateLedger)
			protected.PUT("/ledgers/:id", ledgerHandler.UpdateLedger)
			protected.DELETE("/ledgers/:id", ledgerHandler.DeleteLedger)
			protected.GET("/titles", ledgerHandler.GetTitles)
			protected.POST("/titles", ledgerHandler.CreateTitle)
			protected.PUT("/titles/:id", ledgerHandler.UpdateTitle)
			protected.DELETE("/titles/:id", ledgerHandler.DeleteTitle)

			// Vouchers (Income, Expense, Asset, Liability, Self Transfer, Donation Receipt)
			voucherHandler := voucher.NewVoucherHandler()
			protected.GET("/vouchers", voucherHandler.GetVouchers)
			protected.GET("/vouchers/:id", voucherHandler.GetVoucherByID)
			protected.POST("/vouchers", closedDayCheck, voucherHandler.CreateVoucher)
			protected.POST("/vouchers/:id/approve", middleware.RequireRole(models.RoleAdmin), voucherHandler.ApproveVoucher)
			protected.POST("/vouchers/:id/reject", middleware.RequireRole(models.RoleAdmin), voucherHandler.RejectVoucher)
			protected.POST("/vouchers/:id/cancel", voucherHandler.CancelVoucher)
			protected.GET("/reports/vouchers", voucherHandler.GetVoucherReport)

			// File Uploads (donor docs/photo, bank QR codes, donation/expense attachments)
			uploadHandler := upload.NewUploadHandler()
			protected.POST("/uploads", uploadHandler.UploadFile)

			// Reports
			reportHandler := report.NewReportHandler()
			protected.GET("/reports/summary-book", reportHandler.GetDailySummaryBook)
			protected.GET("/reports/yoy-comparison", reportHandler.GetYoYComparison)
			protected.GET("/reports/yoy-comparison/donors", reportHandler.GetYoYMonthDonors)
			protected.GET("/reports/birthdays", reportHandler.GetBirthdayReport)
		}
	}

	// 7. Start Server
	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	log.Info().Msgf("Backend Server listening on %s (Environment: %s)", serverAddr, cfg.AppEnv)
	if err := r.Run(serverAddr); err != nil {
		log.Fatal().Err(err).Msg("Server crashed")
	}
}

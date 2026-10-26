package main

import (
	"fmt"
	"os"

	"trust-management/backend/internal/attendance"
	"trust-management/backend/internal/auth"
	"trust-management/backend/internal/bank"
	"trust-management/backend/internal/branch"
	"trust-management/backend/internal/cash"
	"trust-management/backend/internal/closing"
	"trust-management/backend/internal/config"
	"trust-management/backend/internal/database"
	"trust-management/backend/internal/donor"
	"trust-management/backend/internal/donation"
	"trust-management/backend/internal/expense"
	"trust-management/backend/internal/expensecategory"
	"trust-management/backend/internal/landing"
	"trust-management/backend/internal/ledger"
	"trust-management/backend/internal/middleware"
	"trust-management/backend/internal/models"
	"trust-management/backend/internal/permissions"
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
	_ = r.SetTrustedProxies(nil)

	// 5. Global Middlewares
	r.Use(gin.Recovery())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.CORSMiddleware(cfg.FrontendURL, cfg.CORSAllowedOrigins))
	// Serve uploaded assets (QR codes, receipt attachments, donor photos)
	// Stored with 128-bit unguessable random filenames (uploading still requires auth)
	r.Static("/uploads", "./uploads")

	// Official Public Trust Website & Compliance Portal (Root Route for Public & Payment Gateway Reviewers)
	r.GET("/", landing.RenderLandingPage)

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
		api.GET("/public/trust-homes", bankHandlerPublic.GetTrustHomes)
		api.GET("/public/schemes/active", schemeHandlerPublic.GetPublicActiveSchemes)
		uploadHandlerPublic := upload.NewUploadHandler()
		api.POST("/public/upload-screenshot", uploadHandlerPublic.UploadFile)
		api.POST("/public/donations/create-order", publicDonationHandler.CreateRazorpayOrder)
		api.POST("/public/donations/create-gateway-order", publicDonationHandler.CreateGatewayOrder)
		api.GET("/public/donations/cashfree-checkout", publicDonationHandler.CashfreeCheckout)
		api.GET("/public/donations/cashfree-return", publicDonationHandler.CashfreeReturn)
		api.POST("/public/donations/verify-and-issue", publicDonationHandler.VerifyAndIssueGatewayDonation)
		api.POST("/public/donations", publicDonationHandler.CreatePublicDonation)
		api.GET("/public/donations/history", publicDonationHandler.GetPublicDonorHistory)
		api.GET("/public/dashboard", publicDonationHandler.GetPublicDashboard)

		// Authenticated Routes Group — standard user JWT check
		protected := api.Group("", middleware.AuthMiddleware(cfg.JWTSecret))
		closedDayCheck := middleware.ClosedDayProtectionMiddleware()
		{
			// Bank Accounts & Trust Homes
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
			protected.GET("/trust-homes", bankHandler.GetTrustHomes)
			protected.POST("/trust-homes", middleware.RequireRole(models.RoleAdmin), bankHandler.CreateTrustHome)
			protected.PUT("/trust-homes/:key", middleware.RequireRole(models.RoleAdmin), bankHandler.UpdateTrustHome)
			protected.DELETE("/trust-homes/:key", middleware.RequireRole(models.RoleAdmin), bankHandler.DeleteTrustHome)

			// Staff & User Management
			userHandler := users.NewUserHandler()
			protected.GET("/users", middleware.RequirePermission("staff:view"), userHandler.GetUsers)
			protected.GET("/users/:id", middleware.RequirePermission("staff:view"), userHandler.GetUserByID)
			protected.POST("/users", middleware.RequirePermission("staff:create"), userHandler.CreateUser)
			protected.PUT("/users/:id", middleware.RequirePermission("staff:edit"), userHandler.UpdateUser)

			// User Action Permissions System
			permHandler := permissions.NewPermissionHandler()
			protected.GET("/permissions", permHandler.GetPermissions)
			protected.GET("/users/:id/permissions", permHandler.GetUserPermissions)
			protected.PUT("/users/:id/permissions", middleware.RequirePermission("staff:permissions"), permHandler.UpdateUserPermissions)

			// Staff Attendance Management
			attendanceHandler := attendance.NewAttendanceHandler()
			protected.GET("/attendance/sheet", middleware.RequirePermission("attendance:mark"), attendanceHandler.GetAttendanceSheet)
			protected.POST("/attendance/batch", middleware.RequirePermission("attendance:mark"), attendanceHandler.BatchSubmitAttendance)
			protected.GET("/attendance/my", attendanceHandler.GetMyAttendance)
			protected.GET("/reports/attendance", middleware.RequirePermission("attendance:reports"), attendanceHandler.GetAttendanceReport)
			protected.GET("/reports/staff-master", middleware.RequirePermission("attendance:reports"), attendanceHandler.GetStaffMasterReport)

			// Branches (Multi-Branch Trust Management)
			branchHandler := branch.NewBranchHandler()
			protected.GET("/branches", branchHandler.GetBranches)
			protected.GET("/branches/license-alerts", middleware.RequirePermission("branch:manage"), branchHandler.GetLicenseAlerts)
			protected.GET("/branches/:id", branchHandler.GetBranchByID)
			protected.POST("/branches", middleware.RequirePermission("branch:manage"), branchHandler.CreateBranch)
			protected.PUT("/branches/:id", middleware.RequirePermission("branch:manage"), branchHandler.UpdateBranch)
			protected.DELETE("/branches/:id", middleware.RequirePermission("branch:manage"), branchHandler.DeleteBranch)

			// Donors
			donorHandler := donor.NewDonorHandler()
			protected.GET("/donors", donorHandler.GetDonors)
			protected.GET("/donors/:id", donorHandler.GetDonorByID)
			protected.POST("/donors", middleware.RequirePermission("donor:create"), donorHandler.CreateDonor)
			protected.PUT("/donors/:id", middleware.RequirePermission("donor:edit"), donorHandler.UpdateDonor)

			// Schemes
			schemeHandler := scheme.NewSchemeHandler()
			protected.GET("/schemes", schemeHandler.GetSchemes)
			protected.GET("/schemes/active", schemeHandler.GetActiveSchemes)
			protected.POST("/schemes", middleware.RequirePermission("scheme:manage"), schemeHandler.CreateScheme)
			protected.POST("/schemes/bulk", middleware.RequirePermission("scheme:manage"), schemeHandler.CreateSchemesBulk)
			protected.PUT("/schemes/:id", middleware.RequirePermission("scheme:manage"), schemeHandler.UpdateScheme)

			// Donations
			donationHandler := donation.NewDonationHandler()
			protected.GET("/donations", donationHandler.GetDonations)
			protected.GET("/donations/pending-verifications", donationHandler.GetPendingDonations)
			protected.GET("/donations/:id", donationHandler.GetDonationByID)
			protected.POST("/donations", closedDayCheck, middleware.RequirePermission("donation:create"), donationHandler.CreateDonation)
			protected.POST("/donations/:id/approve", middleware.RequirePermission("donation:approve"), donationHandler.ApproveDonation)
			protected.POST("/donations/:id/reject", middleware.RequirePermission("donation:reject"), donationHandler.RejectDonation)

			// Cash Management
			cashHandler := cash.NewCashHandler()
			protected.GET("/cash/summary", cashHandler.GetDailyCashSummary)
			protected.POST("/cash/denominations", closedDayCheck, middleware.RequirePermission("cash:submit_denominations"), cashHandler.SubmitCashDenominations)

			// Expenses
			expenseHandler := expense.NewExpenseHandler()
			protected.GET("/expenses", expenseHandler.GetExpenses)
			protected.POST("/expenses", closedDayCheck, middleware.RequirePermission("expense:create"), expenseHandler.CreateExpense)
			protected.POST("/expenses/:id/approve", middleware.RequirePermission("expense:approve"), expenseHandler.ApproveExpense)
			protected.POST("/expenses/:id/reject", middleware.RequirePermission("expense:approve"), expenseHandler.RejectExpense)

			// Expense Categories
			expenseCategoryHandler := expensecategory.NewExpenseCategoryHandler()
			protected.GET("/expense-categories", expenseCategoryHandler.GetExpenseCategories)
			protected.GET("/expense-categories/active", expenseCategoryHandler.GetActiveExpenseCategories)
			protected.POST("/expense-categories", middleware.RequirePermission("expense_category:manage"), expenseCategoryHandler.CreateExpenseCategory)
			protected.PUT("/expense-categories/:id", middleware.RequirePermission("expense_category:manage"), expenseCategoryHandler.UpdateExpenseCategory)

			// Daily Closing State Machine
			closingHandler := closing.NewClosingHandler()
			protected.GET("/closing/status", closingHandler.GetDailyClosingStatus)
			protected.POST("/closing/execute", middleware.RequirePermission("day_closing:execute"), closingHandler.ExecuteDailyClosing)

			// Unlock & Audit
			unlockHandler := unlock.NewUnlockHandler()
			protected.GET("/unlock-requests", unlockHandler.GetUnlockRequests)
			protected.POST("/unlock-requests", middleware.RequirePermission("day_unlock:request"), unlockHandler.SubmitUnlockRequest)
			protected.PUT("/unlock-requests/:id/review", middleware.RequirePermission("day_unlock:review"), unlockHandler.ReviewUnlockRequest)
			protected.GET("/audit-logs", middleware.RequirePermission("audit:view"), unlockHandler.GetAuditLogs)

			// Ledgers & Titles (Chart of Accounts)
			ledgerHandler := ledger.NewLedgerHandler()
			protected.GET("/ledgers", ledgerHandler.GetLedgers)
			protected.POST("/ledgers", middleware.RequirePermission("ledger:manage"), ledgerHandler.CreateLedger)
			protected.PUT("/ledgers/:id", middleware.RequirePermission("ledger:manage"), ledgerHandler.UpdateLedger)
			protected.DELETE("/ledgers/:id", middleware.RequirePermission("ledger:manage"), ledgerHandler.DeleteLedger)
			protected.GET("/titles", ledgerHandler.GetTitles)
			protected.POST("/titles", middleware.RequirePermission("ledger:manage"), ledgerHandler.CreateTitle)
			protected.PUT("/titles/:id", middleware.RequirePermission("ledger:manage"), ledgerHandler.UpdateTitle)
			protected.DELETE("/titles/:id", middleware.RequirePermission("ledger:manage"), ledgerHandler.DeleteTitle)

			// Vouchers (Income, Expense, Asset, Liability, Self Transfer, Donation Receipt)
			voucherHandler := voucher.NewVoucherHandler()
			protected.GET("/vouchers", voucherHandler.GetVouchers)
			protected.GET("/vouchers/:id", voucherHandler.GetVoucherByID)
			protected.POST("/vouchers", closedDayCheck, middleware.RequirePermission("voucher:create"), voucherHandler.CreateVoucher)
			protected.POST("/vouchers/:id/approve", middleware.RequirePermission("voucher:approve"), voucherHandler.ApproveVoucher)
			protected.POST("/vouchers/:id/reject", middleware.RequirePermission("voucher:approve"), voucherHandler.RejectVoucher)
			protected.POST("/vouchers/:id/cancel", middleware.RequirePermission("voucher:cancel"), voucherHandler.CancelVoucher)
			protected.GET("/reports/vouchers", voucherHandler.GetVoucherReport)

			// File Uploads (donor docs/photo, bank QR codes, donation/expense attachments)
			uploadHandler := upload.NewUploadHandler()
			protected.POST("/uploads", uploadHandler.UploadFile)

			// Reports
			reportHandler := report.NewReportHandler()
			protected.GET("/reports/summary-book", reportHandler.GetDailySummaryBook)
			protected.GET("/reports/branches-comparison", middleware.RequireRole(models.RoleAdmin), reportHandler.GetBranchComparisonReport)
			protected.GET("/reports/yoy-comparison", reportHandler.GetYoYComparison)
			protected.GET("/reports/yoy-comparison/donors", reportHandler.GetYoYMonthDonors)
			protected.GET("/reports/birthdays", reportHandler.GetBirthdayReport)
			protected.GET("/reports/staff/:id", middleware.RequirePermission("attendance:reports"), reportHandler.GetStaffAnalyticsReport)
		}
	}

	// 7. Start Server
	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	log.Info().Msgf("Backend Server listening on %s (Environment: %s)", serverAddr, cfg.AppEnv)
	if err := r.Run(serverAddr); err != nil {
		log.Fatal().Err(err).Msg("Server crashed")
	}
}

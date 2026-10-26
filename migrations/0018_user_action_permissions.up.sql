-- Migration 0018: Granular User Action Permissions & Access Control

-- 1. Create master `permissions` registry table
CREATE TABLE IF NOT EXISTS `permissions` (
  `id` INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `code` VARCHAR(60) NOT NULL,
  `module` VARCHAR(40) NOT NULL,
  `name` VARCHAR(100) NOT NULL,
  `description` VARCHAR(255) NOT NULL,
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `uk_permissions_code` (`code`),
  INDEX `idx_permissions_module` (`module`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. Create `user_permissions` join table
CREATE TABLE IF NOT EXISTS `user_permissions` (
  `user_id` BIGINT UNSIGNED NOT NULL,
  `permission_id` INT UNSIGNED NOT NULL,
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`user_id`, `permission_id`),
  INDEX `idx_user_permissions_user` (`user_id`),
  INDEX `idx_user_permissions_permission` (`permission_id`),
  CONSTRAINT `fk_user_permissions_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_user_permissions_permission` FOREIGN KEY (`permission_id`) REFERENCES `permissions` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. Seed initial permissions
INSERT INTO `permissions` (`code`, `module`, `name`, `description`) VALUES
-- Vouchers
('voucher:view', 'VOUCHERS', 'View Vouchers', 'View vouchers, transaction history, and ledger postings'),
('voucher:create', 'VOUCHERS', 'Create Voucher', 'Create payment, receipt, contra, journal, and self-transfer vouchers'),
('voucher:approve', 'VOUCHERS', 'Approve Voucher', 'Authorize or reject pending vouchers created by staff'),
('voucher:cancel', 'VOUCHERS', 'Cancel Voucher', 'Cancel approved or issued vouchers with audit justification'),
('voucher:export', 'VOUCHERS', 'Export & Print Voucher', 'Download PDF voucher copies and export voucher spreadsheets'),

-- Expenses
('expense:view', 'EXPENSES', 'View Expenses', 'View daily operational expenses and vendor payouts'),
('expense:create', 'EXPENSES', 'Add Expense', 'Submit new expense claims with bill/receipt attachments'),
('expense:approve', 'EXPENSES', 'Approve Expense', 'Authorize or reject pending branch expenses'),
('expense_category:manage', 'EXPENSES', 'Manage Expense Categories', 'Create, edit, and deactivate expense categories'),

-- Donations
('donation:view', 'DONATIONS', 'View Donations', 'View donation register and receipt history'),
('donation:create', 'DONATIONS', 'Record Donation', 'Record counter donations and issue formal 80G tax receipts'),
('donation:approve', 'DONATIONS', 'Approve Online Donations', 'Verify and approve pending UPI, gateway, and bank donations'),
('donation:reject', 'DONATIONS', 'Reject Donation', 'Reject suspicious or failed donation entries'),

-- Donors
('donor:view', 'DONORS', 'View Donor Directory', 'Search and view donor demographic and family profiles'),
('donor:create', 'DONORS', 'Register Donor', 'Register new donors into the trust directory'),
('donor:edit', 'DONORS', 'Update Donor', 'Update donor contact, PAN, Aadhaar, and address documents'),

-- Schemes
('scheme:view', 'SCHEMES', 'View Schemes', 'View active meal and pooja schemes and pricing'),
('scheme:manage', 'SCHEMES', 'Manage Schemes', 'Create, update pricing, or deactivate trust schemes'),

-- Treasury & Bank
('bank:view', 'TREASURY', 'View Bank & Cash', 'View bank balances, day summaries, and physical cash status'),
('bank:manage', 'TREASURY', 'Manage Bank Accounts', 'Create, update bank accounts, UPI IDs, and execute transfers'),
('bank:close_day', 'TREASURY', 'Bank Day Closing', 'Perform daily closing verification for individual bank accounts'),
('cash:submit_denominations', 'TREASURY', 'Submit Cash Denominations', 'Count and submit physical denomination currency tallies'),
('day_closing:execute', 'TREASURY', 'Execute Final Day Closing', 'Finalize daily book closing across all registers'),
('day_unlock:request', 'TREASURY', 'Request Day Reopen', 'Request emergency unlock of a closed business date'),
('day_unlock:review', 'TREASURY', 'Review & Approve Unlock', 'Authorize or reject closed date unlock requests'),

-- Staff & Attendance
('staff:view', 'STAFF', 'View Staff Profiles', 'View employee list, contact details, and qualifications'),
('staff:create', 'STAFF', 'Register New Staff', 'Create new staff accounts and login credentials'),
('staff:edit', 'STAFF', 'Update Staff Details', 'Edit employee records, designations, and employment status'),
('staff:reset_password', 'STAFF', 'Reset Staff Passwords', 'Reset login passwords for staff members'),
('staff:permissions', 'STAFF', 'Assign User Permissions', 'Manage and update action-level permissions for any user'),
('attendance:mark', 'STAFF', 'Mark Daily Attendance', 'Record and submit daily staff attendance sheets'),
('attendance:view_my', 'STAFF', 'View My Attendance', 'View own personal monthly attendance records (default for all)'),
('attendance:reports', 'STAFF', 'View Attendance Reports', 'Generate and export staff attendance analytics and demographic reports'),

-- Ledgers & Accounts
('ledger:view', 'LEDGERS', 'View Chart of Accounts', 'View ledger groups, account heads, and opening balances'),
('ledger:manage', 'LEDGERS', 'Manage Ledgers & Titles', 'Create, update, and restructure chart of account titles'),

-- Branches
('branch:view', 'BRANCHES', 'View Branches', 'View branch locations, in-charges, and contact info'),
('branch:manage', 'BRANCHES', 'Manage Branches', 'Create/edit branches and update statutory license renewals'),

-- Reports & Audit
('report:daybook', 'REPORTS', 'View DayBook', 'Access full daily financial DayBook register'),
('report:financial', 'REPORTS', 'View Financial Statements', 'Access Trial Balance, Income & Expenditure, and Balance Sheet'),
('audit:view', 'REPORTS', 'View Audit Logs', 'Access system audit trails, IP logs, and user activity records')
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`), `description`=VALUES(`description`), `module`=VALUES(`module`);

-- 4. Baseline grant for existing STAFF users
INSERT IGNORE INTO `user_permissions` (`user_id`, `permission_id`)
SELECT u.id, p.id FROM `users` u
CROSS JOIN `permissions` p
WHERE u.role = 'STAFF' AND p.code IN (
  'voucher:view', 'voucher:create',
  'expense:view', 'expense:create',
  'donation:view', 'donation:create',
  'donor:view', 'donor:create',
  'scheme:view',
  'bank:view', 'cash:submit_denominations',
  'attendance:view_my',
  'ledger:view',
  'report:daybook'
);

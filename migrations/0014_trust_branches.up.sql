CREATE TABLE IF NOT EXISTS `branches` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `branch_code` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `tamil_name` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `license_number` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `license_issue_date` date DEFAULT NULL,
  `license_expiry_date` date DEFAULT NULL,
  `registration_details` text COLLATE utf8mb4_unicode_ci,
  `incharge_name` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `phone` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `email` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `address_line` text COLLATE utf8mb4_unicode_ci,
  `city` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `state` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'Tamil Nadu',
  `pincode` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `upi_id` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `qr_code_path` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_branches_branch_code` (`branch_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO `branches` (`id`, `branch_code`, `name`, `tamil_name`, `license_number`, `incharge_name`, `city`, `state`, `is_active`)
VALUES (1, 'MAIN', 'Head Office / Main Branch', 'தலைமை அலுவலகம்', 'TRUST/HQ/001', 'Trust Administrator', 'Madurai', 'Tamil Nadu', 1)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

ALTER TABLE `users`
  ADD COLUMN `branch_id` bigint unsigned DEFAULT 1 AFTER `role`,
  ADD KEY `fk_users_branch` (`branch_id`),
  ADD CONSTRAINT `fk_users_branch` FOREIGN KEY (`branch_id`) REFERENCES `branches` (`id`) ON DELETE SET NULL;

ALTER TABLE `vouchers`
  ADD COLUMN `branch_id` bigint unsigned NOT NULL DEFAULT 1 AFTER `id`,
  ADD KEY `fk_vouchers_branch` (`branch_id`),
  ADD CONSTRAINT `fk_vouchers_branch` FOREIGN KEY (`branch_id`) REFERENCES `branches` (`id`) ON DELETE RESTRICT;

ALTER TABLE `expenses`
  ADD COLUMN `branch_id` bigint unsigned NOT NULL DEFAULT 1 AFTER `id`,
  ADD KEY `fk_expenses_branch` (`branch_id`),
  ADD CONSTRAINT `fk_expenses_branch` FOREIGN KEY (`branch_id`) REFERENCES `branches` (`id`) ON DELETE RESTRICT;

ALTER TABLE `donations`
  ADD COLUMN `branch_id` bigint unsigned NOT NULL DEFAULT 1 AFTER `id`,
  ADD KEY `fk_donations_branch` (`branch_id`),
  ADD CONSTRAINT `fk_donations_branch` FOREIGN KEY (`branch_id`) REFERENCES `branches` (`id`) ON DELETE RESTRICT;

ALTER TABLE `cash_transactions`
  ADD COLUMN `branch_id` bigint unsigned NOT NULL DEFAULT 1 AFTER `id`,
  ADD KEY `fk_cash_transactions_branch` (`branch_id`),
  ADD CONSTRAINT `fk_cash_transactions_branch` FOREIGN KEY (`branch_id`) REFERENCES `branches` (`id`) ON DELETE RESTRICT;

ALTER TABLE `bank_accounts`
  ADD COLUMN `branch_id` bigint unsigned DEFAULT NULL AFTER `id`,
  ADD KEY `fk_bank_accounts_branch` (`branch_id`),
  ADD CONSTRAINT `fk_bank_accounts_branch` FOREIGN KEY (`branch_id`) REFERENCES `branches` (`id`) ON DELETE SET NULL;

ALTER TABLE `daily_closings`
  ADD COLUMN `branch_id` bigint unsigned NOT NULL DEFAULT 1 AFTER `id`,
  DROP INDEX `idx_daily_closings_business_date`,
  ADD UNIQUE KEY `idx_daily_closings_date_branch` (`business_date`, `branch_id`),
  ADD KEY `fk_daily_closings_branch` (`branch_id`),
  ADD CONSTRAINT `fk_daily_closings_branch` FOREIGN KEY (`branch_id`) REFERENCES `branches` (`id`) ON DELETE RESTRICT;

-- Baseline schema, generated from the live development database (mysqldump --no-data)
-- after GORM AutoMigrate had created every table used as of this migration.
-- FK checks are disabled during creation so table order doesn't matter.
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `full_name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `email` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `password_hash` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `role` enum('STAFF','ADMIN') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'STAFF',
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`),
  UNIQUE KEY `idx_users_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `donors` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `donor_code` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `full_name` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `phone` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `email` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `address_line` text COLLATE utf8mb4_unicode_ci,
  `city` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `state` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `pincode` varchar(10) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `date_of_birth` date DEFAULT NULL,
  `anniversary_date` date DEFAULT NULL,
  `marital_status` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `aadhaar_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `aadhaar_doc_path` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `pan_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `pan_doc_path` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `notes` text COLLATE utf8mb4_unicode_ci,
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_donors_donor_code` (`donor_code`),
  KEY `idx_donors_phone` (`phone`),
  KEY `idx_donors_city` (`city`),
  KEY `idx_donors_date_of_birth` (`date_of_birth`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `donor_family_members` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `donor_id` bigint unsigned NOT NULL,
  `full_name` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `relationship` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `date_of_birth` date NOT NULL,
  `notes` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_donor_family_members_donor_id` (`donor_id`),
  KEY `idx_donor_family_members_date_of_birth` (`date_of_birth`),
  CONSTRAINT `fk_donors_family_members` FOREIGN KEY (`donor_id`) REFERENCES `donors` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `schemes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `category` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `food_type` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT 'NA',
  `meal_type` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT 'NA',
  `default_amount` decimal(15,2) NOT NULL DEFAULT '0.00',
  `description` text COLLATE utf8mb4_unicode_ci,
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `bank_accounts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `bank_name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `account_number_masked` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `ifsc_code` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `branch` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `opening_balance` decimal(15,2) NOT NULL DEFAULT '0.00',
  `current_balance` decimal(15,2) NOT NULL DEFAULT '0.00',
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `donations` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `donation_number` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `donor_id` bigint unsigned NOT NULL,
  `business_date` date NOT NULL,
  `amount` decimal(15,2) NOT NULL,
  `payment_mode` enum('CASH','BANK') COLLATE utf8mb4_unicode_ci NOT NULL,
  `purpose` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `scheme_id` bigint unsigned DEFAULT NULL,
  `event_type` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `event_person_name` varchar(150) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `family_member_id` bigint unsigned DEFAULT NULL,
  `bank_account_id` bigint unsigned DEFAULT NULL,
  `notes` text COLLATE utf8mb4_unicode_ci,
  `status` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'ACTIVE',
  `created_by_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_donations_donation_number` (`donation_number`),
  KEY `idx_donations_donor_id` (`donor_id`),
  KEY `idx_donations_business_date` (`business_date`),
  KEY `idx_donations_scheme_id` (`scheme_id`),
  KEY `idx_donations_family_member_id` (`family_member_id`),
  KEY `idx_donations_bank_account_id` (`bank_account_id`),
  KEY `fk_donations_created_by` (`created_by_id`),
  CONSTRAINT `fk_donations_created_by` FOREIGN KEY (`created_by_id`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_donations_donor` FOREIGN KEY (`donor_id`) REFERENCES `donors` (`id`),
  CONSTRAINT `fk_donations_scheme` FOREIGN KEY (`scheme_id`) REFERENCES `schemes` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `cash_transactions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `business_date` date NOT NULL,
  `transaction_type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `amount` decimal(15,2) NOT NULL,
  `source_type` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source_id` bigint unsigned NOT NULL,
  `description` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_by_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_cash_transactions_business_date` (`business_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `cash_denominations` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `entity_type` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `entity_id` bigint unsigned NOT NULL,
  `denomination_value` bigint NOT NULL,
  `quantity` bigint NOT NULL DEFAULT '0',
  `total_amount` decimal(15,2) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_cash_denominations_entity_type` (`entity_type`),
  KEY `idx_cash_denominations_entity_id` (`entity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `bank_transactions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `bank_account_id` bigint unsigned NOT NULL,
  `business_date` date NOT NULL,
  `transaction_type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `amount` decimal(15,2) NOT NULL,
  `category` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `reference_number` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `source_type` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source_id` bigint unsigned NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  `created_by_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_bank_transactions_bank_account_id` (`bank_account_id`),
  KEY `idx_bank_transactions_business_date` (`business_date`),
  KEY `idx_bank_transactions_reference_number` (`reference_number`),
  CONSTRAINT `fk_bank_transactions_bank_account` FOREIGN KEY (`bank_account_id`) REFERENCES `bank_accounts` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `expenses` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `expense_number` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `business_date` date NOT NULL,
  `payment_mode` enum('CASH','BANK') COLLATE utf8mb4_unicode_ci NOT NULL,
  `bank_account_id` bigint unsigned DEFAULT NULL,
  `category` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `amount` decimal(15,2) NOT NULL,
  `payee_name` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  `reference_number` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'ACTIVE',
  `created_by_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_expenses_expense_number` (`expense_number`),
  KEY `idx_expenses_business_date` (`business_date`),
  KEY `idx_expenses_bank_account_id` (`bank_account_id`),
  CONSTRAINT `fk_expenses_bank_account` FOREIGN KEY (`bank_account_id`) REFERENCES `bank_accounts` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `daily_closings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `business_date` date NOT NULL,
  `status` enum('OPEN','READY_TO_CLOSE','CLOSED','UNLOCKED') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'OPEN',
  `opening_cash` decimal(15,2) NOT NULL,
  `cash_inflow` decimal(15,2) NOT NULL DEFAULT '0.00',
  `cash_outflow` decimal(15,2) NOT NULL DEFAULT '0.00',
  `expected_closing_cash` decimal(15,2) NOT NULL,
  `physical_cash_count` decimal(15,2) NOT NULL DEFAULT '0.00',
  `cash_difference` decimal(15,2) NOT NULL DEFAULT '0.00',
  `closed_by_id` bigint unsigned DEFAULT NULL,
  `closed_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_daily_closings_business_date` (`business_date`),
  KEY `fk_daily_closings_closed_by` (`closed_by_id`),
  CONSTRAINT `fk_daily_closings_closed_by` FOREIGN KEY (`closed_by_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `bank_closings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `business_date` date NOT NULL,
  `bank_account_id` bigint unsigned NOT NULL,
  `opening_balance` decimal(15,2) NOT NULL,
  `total_credits` decimal(15,2) NOT NULL DEFAULT '0.00',
  `total_debits` decimal(15,2) NOT NULL DEFAULT '0.00',
  `expected_closing` decimal(15,2) NOT NULL,
  `actual_closing` decimal(15,2) NOT NULL,
  `difference` decimal(15,2) NOT NULL DEFAULT '0.00',
  PRIMARY KEY (`id`),
  KEY `idx_bank_closings_business_date` (`business_date`),
  KEY `idx_bank_closings_bank_account_id` (`bank_account_id`),
  CONSTRAINT `fk_bank_closings_bank_account` FOREIGN KEY (`bank_account_id`) REFERENCES `bank_accounts` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `vouchers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `voucher_number` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `voucher_type` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `business_date` date NOT NULL,
  `source_type` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source_id` bigint unsigned NOT NULL,
  `payee_or_donor_name` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `amount` decimal(15,2) NOT NULL,
  `amount_in_words` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `payment_mode` enum('CASH','BANK') COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'ISSUED',
  `created_by_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_vouchers_voucher_number` (`voucher_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `unlock_requests` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `business_date` date NOT NULL,
  `requested_by_id` bigint unsigned NOT NULL,
  `request_reason` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'PENDING',
  `reviewed_by_id` bigint unsigned DEFAULT NULL,
  `review_reason` text COLLATE utf8mb4_unicode_ci,
  `requested_at` datetime(3) DEFAULT NULL,
  `reviewed_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_unlock_requests_business_date` (`business_date`),
  KEY `fk_unlock_requests_requested_by` (`requested_by_id`),
  KEY `fk_unlock_requests_reviewed_by` (`reviewed_by_id`),
  CONSTRAINT `fk_unlock_requests_requested_by` FOREIGN KEY (`requested_by_id`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_unlock_requests_reviewed_by` FOREIGN KEY (`reviewed_by_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `audit_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned DEFAULT NULL,
  `action` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `entity_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `entity_id` bigint unsigned NOT NULL,
  `before_data` json DEFAULT NULL,
  `after_data` json DEFAULT NULL,
  `reason` text COLLATE utf8mb4_unicode_ci,
  `ip_address` varchar(45) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_audit_logs_user_id` (`user_id`),
  KEY `idx_audit_logs_action` (`action`),
  KEY `idx_audit_logs_entity_name` (`entity_name`),
  KEY `idx_audit_logs_entity_id` (`entity_id`),
  CONSTRAINT `fk_audit_logs_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `sequences` (
  `name` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `current_value` bigint NOT NULL DEFAULT '0',
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO `sequences` (`name`, `current_value`) VALUES
  ('DONOR', 0), ('DONATION', 0), ('EXPENSE', 0), ('VOUCHER', 0);

SET FOREIGN_KEY_CHECKS = 1;

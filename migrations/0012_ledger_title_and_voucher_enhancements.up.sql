CREATE TABLE IF NOT EXISTS `ledgers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ledger_no` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `ledger_name` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_by_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ledgers_ledger_no` (`ledger_no`),
  KEY `fk_ledgers_created_by` (`created_by_id`),
  CONSTRAINT `fk_ledgers_created_by` FOREIGN KEY (`created_by_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `voucher_titles` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `title_no` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `title` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `voucher_type` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `ledger_id` bigint unsigned NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_by_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_voucher_titles_title_no` (`title_no`),
  KEY `fk_voucher_titles_ledger` (`ledger_id`),
  KEY `fk_voucher_titles_created_by` (`created_by_id`),
  CONSTRAINT `fk_voucher_titles_ledger` FOREIGN KEY (`ledger_id`) REFERENCES `ledgers` (`id`) ON DELETE RESTRICT,
  CONSTRAINT `fk_voucher_titles_created_by` FOREIGN KEY (`created_by_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `vouchers`
  MODIFY `source_type` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'DIRECT',
  MODIFY `source_id` bigint unsigned NOT NULL DEFAULT 0,
  MODIFY `payee_or_donor_name` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  ADD COLUMN `ledger_id` bigint unsigned DEFAULT NULL AFTER `voucher_type`,
  ADD COLUMN `title_id` bigint unsigned DEFAULT NULL AFTER `ledger_id`,
  ADD COLUMN `bank_account_id` bigint unsigned DEFAULT NULL AFTER `payment_mode`,
  ADD COLUMN `from_bank_account_id` bigint unsigned DEFAULT NULL AFTER `bank_account_id`,
  ADD COLUMN `to_bank_account_id` bigint unsigned DEFAULT NULL AFTER `from_bank_account_id`,
  ADD COLUMN `details` text COLLATE utf8mb4_unicode_ci AFTER `amount_in_words`,
  ADD COLUMN `attachment_path` varchar(255) DEFAULT '' AFTER `details`,
  ADD COLUMN `approved_by_id` bigint unsigned DEFAULT NULL AFTER `created_by_id`,
  ADD COLUMN `approved_at` datetime(3) DEFAULT NULL AFTER `approved_by_id`,
  ADD COLUMN `rejection_reason` varchar(255) DEFAULT '' AFTER `approved_at`,
  ADD CONSTRAINT `fk_vouchers_ledger` FOREIGN KEY (`ledger_id`) REFERENCES `ledgers` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `fk_vouchers_title` FOREIGN KEY (`title_id`) REFERENCES `voucher_titles` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `fk_vouchers_bank` FOREIGN KEY (`bank_account_id`) REFERENCES `bank_accounts` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `fk_vouchers_from_bank` FOREIGN KEY (`from_bank_account_id`) REFERENCES `bank_accounts` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `fk_vouchers_to_bank` FOREIGN KEY (`to_bank_account_id`) REFERENCES `bank_accounts` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `fk_vouchers_approved_by` FOREIGN KEY (`approved_by_id`) REFERENCES `users` (`id`) ON DELETE SET NULL;

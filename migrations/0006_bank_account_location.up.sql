ALTER TABLE `bank_accounts`
  ADD COLUMN `location` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `branch`;

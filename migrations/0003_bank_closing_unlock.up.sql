ALTER TABLE `bank_closings`
  ADD COLUMN `status` varchar(20) NOT NULL DEFAULT 'CLOSED' AFTER `difference`;

ALTER TABLE `unlock_requests`
  ADD COLUMN `entity_type` varchar(20) NOT NULL DEFAULT 'CASH_DAY' AFTER `id`,
  ADD COLUMN `bank_account_id` bigint unsigned DEFAULT NULL AFTER `entity_type`,
  ADD KEY `idx_unlock_requests_bank_account_id` (`bank_account_id`),
  ADD CONSTRAINT `fk_unlock_requests_bank_account` FOREIGN KEY (`bank_account_id`) REFERENCES `bank_accounts` (`id`);

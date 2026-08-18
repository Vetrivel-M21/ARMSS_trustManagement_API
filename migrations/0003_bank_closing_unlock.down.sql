ALTER TABLE `unlock_requests`
  DROP FOREIGN KEY `fk_unlock_requests_bank_account`,
  DROP KEY `idx_unlock_requests_bank_account_id`,
  DROP COLUMN `bank_account_id`,
  DROP COLUMN `entity_type`;

ALTER TABLE `bank_closings`
  DROP COLUMN `status`;

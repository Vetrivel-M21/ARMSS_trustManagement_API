ALTER TABLE `daily_closings`
  DROP FOREIGN KEY `fk_daily_closings_branch`,
  DROP INDEX `idx_daily_closings_date_branch`,
  ADD UNIQUE KEY `idx_daily_closings_business_date` (`business_date`),
  DROP COLUMN `branch_id`;

ALTER TABLE `bank_accounts`
  DROP FOREIGN KEY `fk_bank_accounts_branch`,
  DROP COLUMN `branch_id`;

ALTER TABLE `cash_transactions`
  DROP FOREIGN KEY `fk_cash_transactions_branch`,
  DROP COLUMN `branch_id`;

ALTER TABLE `donations`
  DROP FOREIGN KEY `fk_donations_branch`,
  DROP COLUMN `branch_id`;

ALTER TABLE `expenses`
  DROP FOREIGN KEY `fk_expenses_branch`,
  DROP COLUMN `branch_id`;

ALTER TABLE `vouchers`
  DROP FOREIGN KEY `fk_vouchers_branch`,
  DROP COLUMN `branch_id`;

ALTER TABLE `users`
  DROP FOREIGN KEY `fk_users_branch`,
  DROP COLUMN `branch_id`;

DROP TABLE IF EXISTS `branches`;

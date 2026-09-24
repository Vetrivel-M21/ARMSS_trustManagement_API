ALTER TABLE `vouchers`
  DROP FOREIGN KEY `fk_vouchers_approved_by`,
  DROP FOREIGN KEY `fk_vouchers_to_bank`,
  DROP FOREIGN KEY `fk_vouchers_from_bank`,
  DROP FOREIGN KEY `fk_vouchers_bank`,
  DROP FOREIGN KEY `fk_vouchers_title`,
  DROP FOREIGN KEY `fk_vouchers_ledger`,
  DROP COLUMN `rejection_reason`,
  DROP COLUMN `approved_at`,
  DROP COLUMN `approved_by_id`,
  DROP COLUMN `attachment_path`,
  DROP COLUMN `details`,
  DROP COLUMN `to_bank_account_id`,
  DROP COLUMN `from_bank_account_id`,
  DROP COLUMN `bank_account_id`,
  DROP COLUMN `title_id`,
  DROP COLUMN `ledger_id`;

DROP TABLE IF EXISTS `voucher_titles`;
DROP TABLE IF EXISTS `ledgers`;

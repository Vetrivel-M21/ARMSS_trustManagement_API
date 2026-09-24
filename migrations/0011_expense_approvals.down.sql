ALTER TABLE `expenses`
  DROP FOREIGN KEY `fk_expenses_approved_by`,
  DROP COLUMN `rejection_reason`,
  DROP COLUMN `approved_at`,
  DROP COLUMN `approved_by_id`;

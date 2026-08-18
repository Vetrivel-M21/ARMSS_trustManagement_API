ALTER TABLE `expenses`
  DROP COLUMN `attachment_path`;

ALTER TABLE `donations`
  DROP COLUMN `attachment_path`,
  DROP COLUMN `reference_number`;

ALTER TABLE `bank_accounts`
  DROP COLUMN `qr_code_path`;

ALTER TABLE `donors`
  DROP COLUMN `photo_path`,
  DROP COLUMN `father_name`;

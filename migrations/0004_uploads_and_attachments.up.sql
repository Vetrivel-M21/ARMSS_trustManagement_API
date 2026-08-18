ALTER TABLE `donors`
  ADD COLUMN `father_name` varchar(150) DEFAULT NULL AFTER `full_name`,
  ADD COLUMN `photo_path` varchar(255) DEFAULT NULL AFTER `pan_doc_path`;

ALTER TABLE `bank_accounts`
  ADD COLUMN `qr_code_path` varchar(255) DEFAULT NULL AFTER `opening_balance`;

ALTER TABLE `donations`
  ADD COLUMN `reference_number` varchar(100) DEFAULT NULL AFTER `notes`,
  ADD COLUMN `attachment_path` varchar(255) DEFAULT NULL AFTER `reference_number`;

ALTER TABLE `expenses`
  ADD COLUMN `attachment_path` varchar(255) DEFAULT NULL AFTER `reference_number`;

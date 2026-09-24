ALTER TABLE `bank_accounts`
  ADD COLUMN `upi_id` varchar(100) NOT NULL DEFAULT '' AFTER `qr_code_path`,
  ADD COLUMN `is_app_donation_account` tinyint(1) NOT NULL DEFAULT 0 AFTER `upi_id`;

ALTER TABLE `donations`
  ADD COLUMN `source` varchar(20) NOT NULL DEFAULT 'WEB' AFTER `status`,
  ADD COLUMN `payment_gateway_order_id` varchar(100) DEFAULT '' AFTER `source`,
  ADD COLUMN `payment_gateway_payment_id` varchar(100) DEFAULT '' AFTER `payment_gateway_order_id`,
  ADD COLUMN `verification_status` varchar(30) NOT NULL DEFAULT 'VERIFIED' AFTER `payment_gateway_payment_id`,
  ADD COLUMN `category` varchar(50) NOT NULL DEFAULT 'FOOD' AFTER `verification_status`,
  ADD COLUMN `reason` text AFTER `category`;

ALTER TABLE `bank_transactions`
  ADD COLUMN `source_channel` varchar(30) DEFAULT 'STANDARD' AFTER `source_id`;

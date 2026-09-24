ALTER TABLE `bank_transactions`
  DROP COLUMN `source_channel`;

ALTER TABLE `donations`
  DROP COLUMN `reason`,
  DROP COLUMN `category`,
  DROP COLUMN `verification_status`,
  DROP COLUMN `payment_gateway_payment_id`,
  DROP COLUMN `payment_gateway_order_id`,
  DROP COLUMN `source`;

ALTER TABLE `bank_accounts`
  DROP COLUMN `is_app_donation_account`,
  DROP COLUMN `upi_id`;

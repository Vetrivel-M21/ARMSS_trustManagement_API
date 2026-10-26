-- Migration 0017 Down: Revert staff management and attendance schema

DROP TABLE IF EXISTS `staff_attendances`;

ALTER TABLE `users`
  DROP INDEX `uk_users_staff_code`,
  DROP INDEX `idx_users_employment_status`,
  DROP COLUMN `status_remarks`,
  DROP COLUMN `employment_status`,
  DROP COLUMN `emergency_contact_phone`,
  DROP COLUMN `emergency_contact_name`,
  DROP COLUMN `photo_path`,
  DROP COLUMN `pincode`,
  DROP COLUMN `state`,
  DROP COLUMN `city`,
  DROP COLUMN `address_line`,
  DROP COLUMN `department`,
  DROP COLUMN `designation`,
  DROP COLUMN `date_of_relieving`,
  DROP COLUMN `date_of_joining`,
  DROP COLUMN `date_of_birth`,
  DROP COLUMN `marital_status`,
  DROP COLUMN `education`,
  DROP COLUMN `gender`,
  DROP COLUMN `phone`,
  DROP COLUMN `staff_code`;

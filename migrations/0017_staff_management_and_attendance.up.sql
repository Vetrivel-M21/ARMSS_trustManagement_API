-- Migration 0017: Staff Management, Extended Profiles, Status Tracking, and Manual Attendance System

-- 1. Extend `users` table with Staff profile, demographic, and lifecycle status fields
ALTER TABLE `users`
  ADD COLUMN `staff_code` VARCHAR(30) NULL AFTER `id`,
  ADD COLUMN `phone` VARCHAR(20) NOT NULL DEFAULT '' AFTER `email`,
  ADD COLUMN `gender` VARCHAR(10) NOT NULL DEFAULT 'OTHER' AFTER `phone`,
  ADD COLUMN `education` VARCHAR(150) NOT NULL DEFAULT '' AFTER `gender`,
  ADD COLUMN `marital_status` VARCHAR(20) NOT NULL DEFAULT 'SINGLE' AFTER `education`,
  ADD COLUMN `date_of_birth` DATE NULL AFTER `marital_status`,
  ADD COLUMN `date_of_joining` DATE NULL AFTER `date_of_birth`,
  ADD COLUMN `date_of_relieving` DATE NULL AFTER `date_of_joining`,
  ADD COLUMN `designation` VARCHAR(100) NOT NULL DEFAULT '' AFTER `date_of_relieving`,
  ADD COLUMN `department` VARCHAR(100) NOT NULL DEFAULT '' AFTER `designation`,
  ADD COLUMN `address_line` TEXT NULL AFTER `department`,
  ADD COLUMN `city` VARCHAR(50) NOT NULL DEFAULT '' AFTER `address_line`,
  ADD COLUMN `state` VARCHAR(50) NOT NULL DEFAULT 'Tamil Nadu' AFTER `city`,
  ADD COLUMN `pincode` VARCHAR(10) NOT NULL DEFAULT '' AFTER `state`,
  ADD COLUMN `photo_path` VARCHAR(255) NOT NULL DEFAULT '' AFTER `pincode`,
  ADD COLUMN `emergency_contact_name` VARCHAR(100) NOT NULL DEFAULT '' AFTER `photo_path`,
  ADD COLUMN `emergency_contact_phone` VARCHAR(20) NOT NULL DEFAULT '' AFTER `emergency_contact_name`,
  ADD COLUMN `employment_status` ENUM('ACTIVE', 'RELIEVED', 'TEMPORARILY_BLOCKED') NOT NULL DEFAULT 'ACTIVE' AFTER `role`,
  ADD COLUMN `status_remarks` TEXT NULL AFTER `employment_status`,
  ADD UNIQUE KEY `uk_users_staff_code` (`staff_code`),
  ADD INDEX `idx_users_employment_status` (`employment_status`);

-- 2. Backfill existing users with sequential staff codes
UPDATE `users` SET `staff_code` = CONCAT('STF-', LPAD(id, 4, '0')) WHERE `staff_code` IS NULL OR `staff_code` = '';

-- 3. Create `staff_attendances` table for manual admin attendance logging
CREATE TABLE IF NOT EXISTS `staff_attendances` (
  `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `branch_id` BIGINT UNSIGNED NOT NULL DEFAULT 1,
  `business_date` DATE NOT NULL,
  `status` ENUM('FULL_DAY', 'FIRST_HALF', 'SECOND_HALF', 'ABSENT', 'LEAVE') NOT NULL DEFAULT 'FULL_DAY',
  `remarks` VARCHAR(255) NOT NULL DEFAULT '',
  `marked_by_user_id` BIGINT UNSIGNED NOT NULL,
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `uk_staff_attendance_user_date` (`user_id`, `business_date`),
  INDEX `idx_staff_attendance_date` (`business_date`),
  INDEX `idx_staff_attendance_branch_date` (`branch_id`, `business_date`),
  INDEX `idx_staff_attendance_status` (`status`),
  CONSTRAINT `fk_staff_attendance_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_staff_attendance_branch` FOREIGN KEY (`branch_id`) REFERENCES `branches` (`id`),
  CONSTRAINT `fk_staff_attendance_marker` FOREIGN KEY (`marked_by_user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

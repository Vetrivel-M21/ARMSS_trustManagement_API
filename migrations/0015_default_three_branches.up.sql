-- 1. Safely add upi_id to branches if not present
SET @col_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'branches' AND column_name = 'upi_id');
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `branches` ADD `upi_id` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT \'\'', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 2. Safely add qr_code_path to branches if not present
SET @col_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'branches' AND column_name = 'qr_code_path');
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `branches` ADD `qr_code_path` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT \'\'', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 3. Safely add rejection_reason to donations if not present
SET @col_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'donations' AND column_name = 'rejection_reason');
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `donations` ADD `rejection_reason` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT \'\'', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 4. Safely add verified_by_id to donations if not present
SET @col_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'donations' AND column_name = 'verified_by_id');
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `donations` ADD `verified_by_id` bigint unsigned DEFAULT NULL', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 5. Safely add verified_at to donations if not present
SET @col_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'donations' AND column_name = 'verified_at');
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `donations` ADD `verified_at` datetime(3) DEFAULT NULL', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 6. Update Branch 1 from generic MAIN to Old Age Home
UPDATE `branches`
SET
  `branch_code` = 'OAH',
  `name` = 'Old Age Home',
  `tamil_name` = 'முதியோர்கள் இல்லம்',
  `license_number` = 'DSD/MDU/OAH/2024',
  `incharge_name` = 'Dr. K. Murugan',
  `city` = 'Madurai',
  `state` = 'Tamil Nadu',
  `is_active` = 1
WHERE `id` = 1;

-- 7. Insert or update Branch 2 for Children Home
INSERT INTO `branches` (`id`, `branch_code`, `name`, `tamil_name`, `license_number`, `incharge_name`, `city`, `state`, `is_active`)
VALUES (2, 'CH', 'Children Home', 'குழந்தைகள் இல்லம்', 'DSD/MDU/CH/2024', 'Mrs. S. Meenakshi', 'Madurai', 'Tamil Nadu', 1)
ON DUPLICATE KEY UPDATE
  `branch_code` = VALUES(`branch_code`),
  `name` = VALUES(`name`),
  `tamil_name` = VALUES(`tamil_name`),
  `license_number` = VALUES(`license_number`),
  `incharge_name` = VALUES(`incharge_name`),
  `is_active` = 1;

-- 8. Insert or update Branch 3 for Children Adoption Home
INSERT INTO `branches` (`id`, `branch_code`, `name`, `tamil_name`, `license_number`, `incharge_name`, `city`, `state`, `is_active`)
VALUES (3, 'CAH', 'Children Adoption Home', 'சிறப்பு தத்தெடுத்தல் மையம்', 'DSD/MDU/SAA/2024', 'Dr. R. Anitha', 'Madurai', 'Tamil Nadu', 1)
ON DUPLICATE KEY UPDATE
  `branch_code` = VALUES(`branch_code`),
  `name` = VALUES(`name`),
  `tamil_name` = VALUES(`tamil_name`),
  `license_number` = VALUES(`license_number`),
  `incharge_name` = VALUES(`incharge_name`),
  `is_active` = 1;

-- 9. Sync UPI and QR code paths from trust_home_configs if present
UPDATE `branches` b
JOIN `trust_home_configs` t ON t.home_key = 'OLD_AGE_HOME'
SET b.upi_id = COALESCE(NULLIF(t.upi_id, ''), b.upi_id),
    b.qr_code_path = COALESCE(NULLIF(t.qr_code_path, ''), b.qr_code_path)
WHERE b.id = 1;

UPDATE `branches` b
JOIN `trust_home_configs` t ON t.home_key = 'CHILDREN_HOME'
SET b.upi_id = COALESCE(NULLIF(t.upi_id, ''), b.upi_id),
    b.qr_code_path = COALESCE(NULLIF(t.qr_code_path, ''), b.qr_code_path)
WHERE b.id = 2;

UPDATE `branches` b
JOIN `trust_home_configs` t ON t.home_key = 'ADOPTION_HOME'
SET b.upi_id = COALESCE(NULLIF(t.upi_id, ''), b.upi_id),
    b.qr_code_path = COALESCE(NULLIF(t.qr_code_path, ''), b.qr_code_path)
WHERE b.id = 3;

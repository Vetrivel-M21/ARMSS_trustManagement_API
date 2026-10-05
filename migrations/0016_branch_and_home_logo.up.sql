-- 1. Safely add logo_path to branches if not present
SET @col_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'branches' AND column_name = 'logo_path');
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `branches` ADD `logo_path` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT \'\'', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 2. Safely add logo_path to trust_home_configs if not present
SET @col_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'trust_home_configs' AND column_name = 'logo_path');
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `trust_home_configs` ADD `logo_path` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT \'\'', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 1. Safely remove logo_path from branches if present
SET @col_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'branches' AND column_name = 'logo_path');
SET @sql = IF(@col_exists > 0, 'ALTER TABLE `branches` DROP COLUMN `logo_path`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 2. Safely remove logo_path from trust_home_configs if present
SET @col_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'trust_home_configs' AND column_name = 'logo_path');
SET @sql = IF(@col_exists > 0, 'ALTER TABLE `trust_home_configs` DROP COLUMN `logo_path`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

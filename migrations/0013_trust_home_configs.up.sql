CREATE TABLE IF NOT EXISTS `trust_home_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `home_key` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `home_name` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL,
  `home_name_tamil` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `upi_id` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `qr_code_path` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `description` text COLLATE utf8mb4_unicode_ci,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_trust_home_configs_home_key` (`home_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO `trust_home_configs` (`home_key`, `home_name`, `home_name_tamil`, `upi_id`, `qr_code_path`, `description`, `is_active`)
VALUES
  ('OLD_AGE_HOME', 'Old Age Home', 'முதியோர்கள் இல்லம்', '', '', 'Support elderly residents with nutritious food, healthcare, and dignified shelter.', 1),
  ('CHILDREN_HOME', 'Children Home', 'குழந்தைகள் இல்லம்', '', '', 'Empower underprivileged children with education, nutritious meals, and loving care.', 1),
  ('ADOPTION_HOME', 'Children Adoption Home', 'சிறப்பு தத்தெடுத்தல் மையம்', '', '', 'Specialised adoption center offering safety, medical care, and family placement for infants & children.', 1)
ON DUPLICATE KEY UPDATE `home_name` = VALUES(`home_name`);

ALTER TABLE `donations`
  ADD COLUMN `trust_home` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'OLD_AGE_HOME' AFTER `source`;

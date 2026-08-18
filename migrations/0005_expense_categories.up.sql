CREATE TABLE `expense_categories` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_expense_categories_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO `expense_categories` (`name`, `is_active`, `created_at`, `updated_at`) VALUES
  ('FOOD_EXPENSE', 1, NOW(), NOW()),
  ('SALARY', 1, NOW(), NOW()),
  ('ELECTRICITY', 1, NOW(), NOW()),
  ('PURCHASE', 1, NOW(), NOW()),
  ('BANK_CHARGE', 1, NOW(), NOW()),
  ('MAINTENANCE', 1, NOW(), NOW()),
  ('TRANSPORT', 1, NOW(), NOW()),
  ('OTHER', 1, NOW(), NOW());

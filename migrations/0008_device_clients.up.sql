CREATE TABLE `device_clients` (
  `id` varchar(36) NOT NULL,
  `name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `client_secret_hash` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `last_used_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Pre-registered client identity for the MIS Desktop app's Trust Portal tab.
-- The plaintext secret is held in mis_desktop's config (lib/core/config/trust_app_client.dart) —
-- only its bcrypt hash is stored here.
INSERT INTO `device_clients` (`id`, `name`, `client_secret_hash`, `is_active`, `created_at`) VALUES
  ('b1880c9d-0792-47fc-8200-8d1d2d74c96c', 'MIS Desktop - Trust Portal', '$2a$10$IXoN4MgSvD4AKZXyhqUq8uSZWmuE7qG9MqWUgjgcTD0XqcB5RzYa.', 1, NOW());

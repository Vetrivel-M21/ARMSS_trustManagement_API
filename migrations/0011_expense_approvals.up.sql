ALTER TABLE `expenses`
  ADD COLUMN `approved_by_id` bigint unsigned DEFAULT NULL AFTER `created_by_id`,
  ADD COLUMN `approved_at` datetime(3) DEFAULT NULL AFTER `approved_by_id`,
  ADD COLUMN `rejection_reason` varchar(255) NOT NULL DEFAULT '' AFTER `approved_at`,
  ADD CONSTRAINT `fk_expenses_approved_by` FOREIGN KEY (`approved_by_id`) REFERENCES `users` (`id`) ON DELETE SET NULL;

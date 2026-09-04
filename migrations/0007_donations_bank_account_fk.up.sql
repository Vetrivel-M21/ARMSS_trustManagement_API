UPDATE `donations` d
LEFT JOIN `bank_accounts` b ON d.bank_account_id = b.id
SET d.bank_account_id = NULL
WHERE d.bank_account_id IS NOT NULL AND b.id IS NULL;

ALTER TABLE `donations`
  ADD CONSTRAINT `fk_donations_bank_account` FOREIGN KEY (`bank_account_id`) REFERENCES `bank_accounts` (`id`);

-- Revert to Main branch if rolled back
UPDATE `branches`
SET `branch_code` = 'MAIN', `name` = 'Head Office / Main Branch', `tamil_name` = 'தலைமை அலுவலகம்'
WHERE `id` = 1;

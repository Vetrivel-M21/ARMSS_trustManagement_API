ALTER TABLE `donations`
  ADD COLUMN `event_date` date DEFAULT NULL AFTER `event_person_name`,
  ADD COLUMN `relationship_to_donor` varchar(30) DEFAULT NULL AFTER `event_date`;

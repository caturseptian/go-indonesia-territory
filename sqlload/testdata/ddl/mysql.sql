CREATE TABLE `territory_provinces` (
  `code` VARCHAR(2) NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `latitude` DECIMAL(9,6),
  `longitude` DECIMAL(9,6),
  `spatial_status` VARCHAR(16),
  `bbox` JSON,
  PRIMARY KEY (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `territory_regencies` (
  `code` VARCHAR(5) NOT NULL,
  `province_code` VARCHAR(2) NOT NULL,
  `type` VARCHAR(9) NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `latitude` DECIMAL(9,6),
  `longitude` DECIMAL(9,6),
  `spatial_status` VARCHAR(16),
  `bbox` JSON,
  PRIMARY KEY (`code`),
  FOREIGN KEY (`province_code`) REFERENCES `territory_provinces` (`code`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX `territory_regencies_province_code_idx` ON `territory_regencies` (`province_code`);

CREATE INDEX `territory_regencies_name_idx` ON `territory_regencies` (`name`);

CREATE TABLE `territory_districts` (
  `code` VARCHAR(8) NOT NULL,
  `regency_code` VARCHAR(5) NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `latitude` DECIMAL(9,6),
  `longitude` DECIMAL(9,6),
  `spatial_status` VARCHAR(16),
  `bbox` JSON,
  PRIMARY KEY (`code`),
  FOREIGN KEY (`regency_code`) REFERENCES `territory_regencies` (`code`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX `territory_districts_regency_code_idx` ON `territory_districts` (`regency_code`);

CREATE INDEX `territory_districts_name_idx` ON `territory_districts` (`name`);

CREATE TABLE `territory_villages` (
  `code` VARCHAR(13) NOT NULL,
  `district_code` VARCHAR(8) NOT NULL,
  `type` VARCHAR(9) NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `postal_code` VARCHAR(5),
  `latitude` DECIMAL(9,6),
  `longitude` DECIMAL(9,6),
  `spatial_status` VARCHAR(16),
  `bbox` JSON,
  PRIMARY KEY (`code`),
  FOREIGN KEY (`district_code`) REFERENCES `territory_districts` (`code`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX `territory_villages_district_code_idx` ON `territory_villages` (`district_code`);

CREATE INDEX `territory_villages_name_idx` ON `territory_villages` (`name`);

CREATE INDEX `territory_villages_postal_code_idx` ON `territory_villages` (`postal_code`);

CREATE TABLE `territory_islands` (
  `code` VARCHAR(11) NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `province_code` VARCHAR(2) NOT NULL,
  `regency_code` VARCHAR(5),
  `latitude` DECIMAL(9,6),
  `longitude` DECIMAL(9,6),
  `area_km2` DECIMAL(12,4),
  `inhabited` TINYINT(1),
  PRIMARY KEY (`code`),
  FOREIGN KEY (`province_code`) REFERENCES `territory_provinces` (`code`) ON DELETE CASCADE,
  FOREIGN KEY (`regency_code`) REFERENCES `territory_regencies` (`code`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX `territory_islands_province_code_idx` ON `territory_islands` (`province_code`);

CREATE INDEX `territory_islands_regency_code_idx` ON `territory_islands` (`regency_code`);

CREATE INDEX `territory_islands_name_idx` ON `territory_islands` (`name`);

CREATE TABLE `territory_code_changes` (
  `seq` INTEGER NOT NULL,
  `from_document` VARCHAR(64) NOT NULL,
  `to_document` VARCHAR(64) NOT NULL,
  `level` VARCHAR(8) NOT NULL,
  `change` VARCHAR(8) NOT NULL,
  `code` VARCHAR(13) NOT NULL,
  `old_name` VARCHAR(255),
  `new_name` VARCHAR(255),
  PRIMARY KEY (`seq`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX `territory_code_changes_code_idx` ON `territory_code_changes` (`code`);

CREATE TABLE `territory_meta` (
  `key` VARCHAR(64) NOT NULL,
  `value` VARCHAR(255) NOT NULL,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

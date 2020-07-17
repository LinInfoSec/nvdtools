CREATE DATABASE IF NOT EXISTS lininfosec;

USE lininfosec;

DROP TABLE IF EXISTS
	`cves_notified`,
	`cpe_monitored`,
	`cpe_references`,
	`cpe_dict`
;

CREATE TABLE `cpe_dict` (
	`uri`        VARCHAR(255) NOT NULL, 
	`part`       CHAR NOT NULL,
	`vendor`     VARCHAR(128) NOT NULL,
	`product`    VARCHAR(128) NOT NULL,
	`version`    VARCHAR(128),
	`updatecl`   VARCHAR(128),
	`edition`    VARCHAR(128),
	`swedition`  VARCHAR(128), `TargetSW`   VARCHAR(128),
	`targethw`   VARCHAR(128),
	`other`      VARCHAR(128),
	`language`   VARCHAR(128),
	`title`      TEXT ,
	PRIMARY KEY (`uri`),
	KEY (`part`, `vendor`, `product`),
	FULLTEXT INDEX (`title`)
)
ENGINE InnoDB
DEFAULT CHARACTER SET utf8mb4
COMMENT 'CPE dictionnary'
;

CREATE TABLE `cpe_references` (
	`cpe_uri` VARCHAR(255) NOT NULL,
	`url` TEXT NOT NULL,
	`description` TEXT,
	CONSTRAINT reference_fkey 
		FOREIGN KEY (`cpe_uri`) REFERENCES cpe_dict (`uri`)
		ON DELETE CASCADE
		ON UPDATE RESTRICT
)
ENGINE InnoDB
DEFAULT CHARACTER SET utf8mb4
COMMENT 'References for each cpe in the CPE dictionnary'
;



CREATE TABLE `cves_notified` (
	`id`                   INT          NOT NULL AUTO_INCREMENT COMMENT 'ID of the notification',
	`ts`                   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Time of the notification',
	`cve_id`               VARCHAR(128) NOT NULL COMMENT 'Common Vulnerability and Exposure (CVE) ID',
	`cpe_uri`              VARCHAR(255) NOT NULL COMMENT 'The cpe for which a notification has been sent',
	PRIMARY KEY (`id`),
	CONSTRAINT cve_notified_fkey
		FOREIGN KEY (`cpe_uri`) REFERENCES cpe_dict (`uri`)
		ON DELETE CASCADE
		ON UPDATE RESTRICT
)
ENGINE InnoDB
DEFAULT CHARACTER SET utf8mb4
COMMENT 'Notification history'
;

CREATE TABLE cpe_monitored (
	`cpe_uri` VARCHAR(255) NOT NULL,
	CONSTRAINT monitored_fkey
		FOREIGN KEY (`cpe_uri`) REFERENCES cpe_dict (`uri`)
		ON DELETE CASCADE
		ON UPDATE RESTRICT
)
ENGINE InnoDB
DEFAULT CHARACTER SET utf8mb4
COMMENT 'CPEs to be monitored for new CVE publications'
;

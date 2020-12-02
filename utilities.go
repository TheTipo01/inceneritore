package main

import (
	"database/sql"
	"log"
)

const (
	tblConfig     = "CREATE TABLE IF NOT EXISTS `config` (  `serverId` varchar(18) NOT NULL,  `serverName` varchar(100) NOT NULL,  `ruolo` varchar(18) NOT NULL,  `testuale` varchar(18) NOT NULL,  `vocale` varchar(18) NOT NULL,  `invito` varchar(30) NOT NULL,  PRIMARY KEY (`serverId`),  CONSTRAINT `FK_config_server` FOREIGN KEY (`serverId`) REFERENCES `server` (`id`));"
	tblInceneriti = "CREATE TABLE IF NOT EXISTS `inceneriti` (  `id` int(11) NOT NULL AUTO_INCREMENT,  `UserID` varchar(18) NOT NULL,  `TimeStamp` datetime NOT NULL,  `serverId` varchar(18) NOT NULL,  PRIMARY KEY (`id`),  KEY `FK_inceneriti_utenti` (`UserID`),  KEY `FK_inceneriti_config` (`serverId`),  CONSTRAINT `FK_inceneriti_config` FOREIGN KEY (`serverId`) REFERENCES `config` (`serverId`),  CONSTRAINT `FK_inceneriti_utenti` FOREIGN KEY (`UserId`) REFERENCES `utenti` (`UserID`));"
	tblRoles      = "CREATE TABLE IF NOT EXISTS `roles` (  `UserID` varchar(18) NOT NULL,  `server` varchar(18) NOT NULL,  `Roles` text NOT NULL,  `Nickname` varchar(32) NOT NULL,  PRIMARY KEY (`UserID`),  KEY `FK_roles_config` (`server`),  CONSTRAINT `FK__utenti` FOREIGN KEY (`UserID`) REFERENCES `utenti` (`UserID`),  CONSTRAINT `FK_roles_config` FOREIGN KEY (`server`) REFERENCES `config` (`serverId`));"
	tblUtenti     = "CREATE TABLE IF NOT EXISTS `utenti` (  `UserID` varchar(18) NOT NULL,  `Name` varchar(32) NOT NULL,  PRIMARY KEY (`UserID`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
)

func execQuery(query string, db *sql.DB) {
	statement, err := db.Prepare(query)
	if err != nil {
		log.Println("Error preparing query,", err)
		return
	}

	_, err = statement.Exec()
	if err != nil {
		log.Println("Error creating table,", err)
	}
}

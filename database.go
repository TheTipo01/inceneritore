package main

import (
	"github.com/bwmarrin/lit"
	"time"
)

const (
	tblConfig     = "CREATE TABLE IF NOT EXISTS `config` (  `serverId` varchar(18) NOT NULL,  `serverName` varchar(100) NOT NULL,  `ruolo` varchar(18) NOT NULL,  `testuale` varchar(18) NOT NULL,  `vocale` varchar(18) NOT NULL,  `invito` varchar(30) NOT NULL,  `message` varchar(2000) NOT NULL,   PRIMARY KEY (`serverId`),  CONSTRAINT `FK_config_server` FOREIGN KEY (`serverId`) REFERENCES `server` (`id`));"
	tblInceneriti = "CREATE TABLE IF NOT EXISTS `inceneriti` (  `id` int(11) NOT NULL AUTO_INCREMENT,  `UserID` varchar(18) NOT NULL,  `TimeStamp` datetime NOT NULL,  `serverId` varchar(18) NOT NULL,  PRIMARY KEY (`id`),  KEY `FK_inceneriti_utenti` (`UserID`),  KEY `FK_inceneriti_config` (`serverId`),  CONSTRAINT `FK_inceneriti_config` FOREIGN KEY (`serverId`) REFERENCES `config` (`serverId`),  CONSTRAINT `FK_inceneriti_utenti` FOREIGN KEY (`UserId`) REFERENCES `utenti` (`UserID`));"
	tblRoles      = "CREATE TABLE IF NOT EXISTS `roles` (  `UserID` varchar(18) NOT NULL,  `server` varchar(18) NOT NULL,  `Roles` text NOT NULL,  `Nickname` varchar(32) NOT NULL,  PRIMARY KEY (`UserID`),  KEY `FK_roles_config` (`server`),  CONSTRAINT `FK__utenti` FOREIGN KEY (`UserID`) REFERENCES `utenti` (`UserID`),  CONSTRAINT `FK_roles_config` FOREIGN KEY (`server`) REFERENCES `config` (`serverId`));"
	tblUtenti     = "CREATE TABLE IF NOT EXISTS `utenti` (  `UserID` varchar(18) NOT NULL,  `Name` varchar(32) NOT NULL,  PRIMARY KEY (`UserID`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
)

// Executes a simple query
func execQuery(query string) {
	stm, err := db.Prepare(query)
	if err != nil {
		lit.Error("Error preparing query, %s", err)
		return
	}

	_, err = stm.Exec()
	if err != nil {
		lit.Error("Error creating table, %s", err)
	}

	_ = stm.Close()
}

// Loads Config for all the servers
func loadConfig() {
	var (
		serverID string
		ruolo    string
		testuale string
		vocale   string
		invito   string
		nome     string
		message  string
	)

	rows, err := db.Query("SELECT * FROM config")
	if err != nil {
		lit.Error("Error querying db, %s", err)
	}

	for rows.Next() {
		err = rows.Scan(&serverID, &nome, &ruolo, &testuale, &vocale, &invito, &message)
		if err != nil {
			lit.Error("Error scanning rows from query, %s", err)
			continue
		}

		config[serverID] = Config{
			ruolo:    ruolo,
			testuale: testuale,
			vocale:   vocale,
			invito:   invito,
			nome:     nome,
			lastKick: make(map[string]*time.Time),
			messagge: message,
		}
	}
}

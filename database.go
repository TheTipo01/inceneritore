package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	"strings"
)

const (
	tblConfig     = "CREATE TABLE IF NOT EXISTS `config` ( `serverId` varchar(20) NOT NULL, `serverName` varchar(100) NOT NULL, `ruolo` varchar(20) NOT NULL, `testuale` varchar(20) NOT NULL, `vocale` varchar(20) NOT NULL, `invito` varchar(30) NOT NULL, `message` varchar(2000) NOT NULL, `boosterRole` varchar(20) NOT NULL, PRIMARY KEY (`serverId`) ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
	tblInceneriti = "CREATE TABLE IF NOT EXISTS `inceneriti` ( `id` int(11) NOT NULL AUTO_INCREMENT, `UserID` varchar(20) NOT NULL, `TimeStamp` datetime NOT NULL DEFAULT current_timestamp(), `serverId` varchar(20) NOT NULL, PRIMARY KEY (`id`), KEY `FK_inceneriti_utenti` (`UserID`), KEY `FK_inceneriti_config` (`serverId`), CONSTRAINT `FK_inceneriti_config` FOREIGN KEY (`serverId`) REFERENCES `config` (`serverId`) ON DELETE NO ACTION ON UPDATE NO ACTION, CONSTRAINT `FK_inceneriti_utenti` FOREIGN KEY (`UserID`) REFERENCES `utenti` (`UserID`) ON DELETE NO ACTION ON UPDATE NO ACTION ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
	tblRoles      = "CREATE TABLE IF NOT EXISTS `roles` ( `UserID` varchar(20) NOT NULL, `server` varchar(20) NOT NULL, `Roles` text NOT NULL, `Nickname` varchar(32) NOT NULL, PRIMARY KEY (`UserID`,`server`), KEY `FK_roles_config` (`server`), CONSTRAINT `FK_roles_config` FOREIGN KEY (`server`) REFERENCES `config` (`serverId`) ON DELETE NO ACTION ON UPDATE NO ACTION, CONSTRAINT `FK_roles_utenti` FOREIGN KEY (`UserID`) REFERENCES `utenti` (`UserID`) ON DELETE NO ACTION ON UPDATE NO ACTION ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
	tblUtenti     = "CREATE TABLE IF NOT EXISTS `utenti` ( `UserID` varchar(20) NOT NULL, `Name` varchar(32) NOT NULL, PRIMARY KEY (`UserID`) ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
)

// Executes a simple query
func execQuery(query ...string) {
	for _, q := range query {
		_, err := db.Exec(q)
		if err != nil {
			lit.Error("Error creating table, %s", err)
		}
	}
}

// Loads Config for all the servers
func loadConfig() {
	rows, err := db.Query("SELECT * FROM config")
	if err != nil {
		lit.Error("Error querying db, %s", err)
	}

	for rows.Next() {
		var (
			s        Server
			serverID string
		)

		err = rows.Scan(&serverID, &s.nome, &s.ruolo, &s.testuale, &s.vocale, &s.invito, &s.messagge, &s.boostRole)
		if err != nil {
			lit.Error("Error scanning rows from query, %s", err)
			continue
		}

		config[serverID] = s
	}
}

// Returns the number of times a user has been kicked
func getIncenerimenti(userID string, guildID string) int {
	var n int

	_ = db.QueryRow("SELECT COUNT(*) FROM inceneriti WHERE UserID=? AND serverId=?", userID, guildID).Scan(&n)

	return n
}

// Saves roles of a user
func saveRoles(m *discordgo.Member, guildID string) {
	// Remove the booster role if the user has it
	for i, v := range m.Roles {
		if v == config[guildID].boostRole {
			m.Roles = append(m.Roles[:i], m.Roles[i+1:]...)
			break
		}
	}

	roles := strings.Join(m.Roles, ",")

	// User
	_, err := db.Exec("INSERT IGNORE INTO utenti (UserID, Name) VALUES (?, ?)", m.User.ID, m.User.Username)
	if err != nil {
		lit.Error("Error inserting into the db, %s", err)
	}

	// Role
	_, err = db.Exec("INSERT INTO roles (UserID, server, Roles, Nickname) VALUES (?, ?, ?, ?)", m.User.ID, guildID, roles, m.Nick)
	if err != nil {
		lit.Error("Error inserting into the db, %s", err)
	}
}

package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	cmap "github.com/orcaman/concurrent-map/v2"

	"strings"
)

const (
	tblIncinerated = "CREATE TABLE `incinerated` ( `id` INT(11) NOT NULL AUTO_INCREMENT, `userID` VARCHAR(20) NOT NULL COLLATE 'utf8mb4_general_ci', `timestamp` DATETIME NOT NULL DEFAULT current_timestamp(), `serverID` VARCHAR(20) NOT NULL COLLATE 'utf8mb4_general_ci', PRIMARY KEY (`id`) USING BTREE, INDEX `FK_incinerated_users` (`userID`) USING BTREE, INDEX `FK_incinerated_servers` (`serverID`) USING BTREE, CONSTRAINT `FK_incinerated_servers` FOREIGN KEY (`serverID`) REFERENCES `servers` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT `FK_incinerated_users` FOREIGN KEY (`userID`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION ) COLLATE='utf8mb4_general_ci' ENGINE=InnoDB ;"
	tblRoles       = "CREATE TABLE `roles` ( `userID` VARCHAR(20) NOT NULL COLLATE 'utf8mb4_general_ci', `serverID` VARCHAR(20) NOT NULL COLLATE 'utf8mb4_general_ci', `roles` TEXT NOT NULL COLLATE 'utf8mb4_general_ci', `nickname` VARCHAR(32) NOT NULL COLLATE 'utf8mb4_general_ci', PRIMARY KEY (`userID`, `serverID`) USING BTREE, INDEX `FK_roles_servers` (`serverID`) USING BTREE, INDEX `userID` (`userID`) USING BTREE, CONSTRAINT `FK_roles_servers` FOREIGN KEY (`serverID`) REFERENCES `servers` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT `FK_roles_users` FOREIGN KEY (`userID`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION ) COLLATE='utf8mb4_general_ci' ENGINE=InnoDB ;"
	tblServers     = "CREATE TABLE `servers` ( `id` VARCHAR(20) NOT NULL COLLATE 'utf8mb4_general_ci', `name` VARCHAR(100) NOT NULL COLLATE 'utf8mb4_general_ci', PRIMARY KEY (`id`) USING BTREE ) COLLATE='utf8mb4_general_ci' ENGINE=InnoDB ;"
	tblUsers       = "CREATE TABLE `users` ( `id` VARCHAR(20) NOT NULL COLLATE 'utf8mb4_general_ci', `name` VARCHAR(32) NOT NULL COLLATE 'utf8mb4_general_ci', `isBot` TINYINT(1) UNSIGNED NOT NULL DEFAULT '0', PRIMARY KEY (`id`) USING BTREE ) COLLATE='utf8mb4_general_ci' ENGINE=InnoDB ;"
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

// Returns the number of times a user has been kicked
func getIncenerimenti(userID string, guildID string) int {
	var n int

	_ = db.QueryRow("SELECT COUNT(*) FROM incinerated WHERE userID=? AND serverID=?", userID, guildID).Scan(&n)

	return n
}

// Saves roles of a user
func saveRoles(m *discordgo.Member, guildID string) error {
	// Remove the booster role if the user has it
	c, _ := config.Get(guildID)

	for i, v := range m.Roles {
		if v == c.BoostRole {
			m.Roles = append(m.Roles[:i], m.Roles[i+1:]...)
			break
		}
	}

	roles := strings.Join(m.Roles, ",")

	// Role
	_, err := db.Exec("INSERT INTO roles (userID, serverID, roles, nickname) VALUES (?, ?, ?, ?)", m.User.ID, guildID, roles, m.Nick)
	return err
}

// Adds the user to the db, to show stats on the website
func insertIncenerimento(UserID string, serverID string) {
	_, err := db.Exec("INSERT INTO incinerated (userID, timestamp, serverID) VALUES (?, NOW(), ?)", UserID, serverID)
	if err != nil {
		lit.Error("Error inserting into the db, %s", err)
	}
}

func loadIsBot() *cmap.ConcurrentMap[string, bool] {
	c := cmap.New[bool]()

	rows, err := db.Query("SELECT id, isBot FROM users")
	if err != nil {
		lit.Error("Error querying db, %s", err)
	}

	for rows.Next() {
		var (
			userID string
			isBot  bool
		)

		err = rows.Scan(&userID, &isBot)
		if err != nil {
			lit.Error("Error scanning rows from query, %s", err)
			continue
		}

		c.Set(userID, isBot)
	}

	return &c
}

// Saves if a user is a bot or not
func saveIsBot(userID, name string, isBot bool) error {
	_, err := db.Exec("INSERT INTO users (id, name, isBot) VALUES (?, ?, ?)", userID, name, isBot)

	return err
}

// Save the server name if it doesn't exist, or update it if it does
func saveServer(id, name string) error {
	_, err := db.Exec("INSERT INTO servers (id, name) VALUES (?, ?) ON DUPLICATE KEY UPDATE name=?", id, name, name)
	return err
}

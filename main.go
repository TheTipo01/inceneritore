package main

import (
	"database/sql"
	"github.com/bwmarrin/lit"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	token  string
	config = make(map[string]Config)
	db     *sql.DB
)

func init() {
	lit.LogLevel = lit.LogError

	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			lit.Error("Config file not found! See example_config.yml")
			return
		}
	} else {
		// Config file found
		token = viper.GetString("token")

		db, err = sql.Open(viper.GetString("drivername"), viper.GetString("datasourcename"))
		if err != nil {
			lit.Error("Error opening database connection, %s", err)
			return
		}

		// Set lit.LogLevel to the given value
		switch strings.ToLower(viper.GetString("loglevel")) {
		case "logerror", "error":
			lit.LogLevel = lit.LogError
			break
		case "logwarning", "warning":
			lit.LogLevel = lit.LogWarning
			break
		case "loginformational", "informational":
			lit.LogLevel = lit.LogInformational
			break
		case "logdebug", "debug":
			lit.LogLevel = lit.LogDebug
			break
		}

		// Creates all the tables
		execQuery(tblInceneriti)
		execQuery(tblRoles)
		execQuery(tblUtenti)
		execQuery(tblConfig)

		// And loads the config for all of the servers
		loadConfig()
	}
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		lit.Error("Error creating Discord session, %s", err)
		return
	}

	dg.AddHandler(voiceStateUpdate)
	dg.AddHandler(messageCreate)
	dg.AddHandler(guildMemberAdd)
	dg.AddHandler(ready)

	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMembers | discordgo.IntentsGuildVoiceStates | discordgo.IntentsGuildMessages)

	err = dg.Open()
	if err != nil {
		lit.Error("Error opening connection, %s", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	lit.Info("inceneritore is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	_ = dg.Close()

	// And the db
	_ = db.Close()
}

func ready(s *discordgo.Session, _ *discordgo.Ready) {
	// Set the playing status.
	err := s.UpdateGameStatus(0, "inceneritore.ga")
	if err != nil {
		lit.Error("Can't set status, %s", err)
	}
}

// Chiamata quando qualcuno entra o viene spostato in un canale vocale
func voiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	// Checks if the voice state update is from the correct channel and the user isn't a bot
	if user, err := s.User(v.UserID); err == nil && (v.ChannelID != config[v.GuildID].vocale || user.Bot) {
		return
	}

	if config[v.GuildID].lastKick[v.UserID] != nil && time.Now().Sub(*config[v.GuildID].lastKick[v.UserID]) < time.Second {
		lit.Warn("Event fired twice")
		return
	}

	currentTime := time.Now()
	config[v.GuildID].lastKick[v.UserID] = &currentTime

	m, err := s.GuildMember(v.GuildID, v.UserID)
	if err != nil {
		lit.Error("Error creating member, %s", err)
	}

	saveRoles(m, v.GuildID)

	// Add the role, so the user doesn't move
	err = s.GuildMemberEdit(v.GuildID, v.UserID, []string{config[v.GuildID].ruolo})
	if err != nil {
		lit.Error("Error adding role, %s", err)

		removeRole(v.UserID, v.GuildID)
		return
	}

	// Wait 3 seconds
	time.Sleep(3 * time.Second)

	// Search for the user private message channel
	canale, err := s.UserChannelCreate(v.UserID)
	if err != nil {
		lit.Error("Error getting DM channel id, %s", err)

		_ = s.GuildMemberRoleRemove(v.GuildID, v.UserID, config[v.GuildID].ruolo)

		removeRole(v.UserID, v.GuildID)
		return
	}

	// Send the invite link
	_, err = s.ChannelMessageSend(canale.ID, config[v.GuildID].invito)
	if err != nil {
		lit.Error("Error sending message, %s", err)
	}

	// And kicks the user
	err = s.GuildMemberDelete(v.GuildID, v.UserID)
	if err != nil {
		lit.Error("Error kicking user, %s", err)

		//Se non riesco tolgo il ruolo
		_ = s.GuildMemberRoleRemove(v.GuildID, v.UserID, config[v.GuildID].ruolo)

		removeRole(v.UserID, v.GuildID)
		return
	}

	// Tracks when the user was kicked, to show on the website
	insertionUser(v.UserID, v.GuildID)
	// And sends the message on the guild text channel
	sendMessage(s, v.UserID, v.GuildID)
}

// Used to generate scoreboard
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Content != "?inceneriti" || m.Author.Bot || m.Author.ID == s.State.User.ID {
		return
	}

	var (
		name string
		mex  string
		cont int
	)

	// Querying db
	rows, err := db.Query("SELECT Name, Count(inceneriti.UserID) FROM inceneriti, utenti WHERE utenti.UserID = inceneriti.UserID AND serverID=? GROUP BY inceneriti.UserID ORDER BY Count(inceneriti.UserID) DESC", m.GuildID)
	if err != nil {
		lit.Error("Error querying db, %s", err)
	}

	for rows.Next() {
		err = rows.Scan(&name, &cont)
		if err != nil {
			lit.Error("Error scanning rows from query, %s", err)
			continue
		}

		mex += name + " - " + strconv.Itoa(cont) + "\n\n"
	}

	_, err = s.ChannelMessageSend(m.ChannelID, mex)
	if err != nil {
		lit.Error("Error sending message, %s", err)
	}
}

// Used to add roles&nick back to the user
func guildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	var roles, nickname string

	err := db.QueryRow("SELECT Roles, Nickname FROM roles WHERE UserID=? AND server=?", m.User.ID, m.GuildID).Scan(&roles, &nickname)
	if err != nil {
		lit.Error("Error scanning row from query, %s", err)
	}

	stm, _ := db.Prepare("DELETE FROM roles WHERE UserID=? AND server=?")

	_, err = stm.Exec(m.User.ID, m.GuildID)
	if err != nil {
		lit.Error("Error deleting from db, %s", err)
	}

	_ = stm.Close()

	err = s.GuildMemberNickname(m.GuildID, m.User.ID, nickname)
	if err != nil {
		lit.Error("Error changing nickname, %s", err)
	}

	for _, role := range strings.Split(roles, ",") {
		if role != config[m.GuildID].ruolo && role != "" {
			err = s.GuildMemberRoleAdd(m.GuildID, m.User.ID, role)
			if err != nil {
				lit.Error("Error adding role, %s", err)
			}
		}
	}
}

// Adds the user to the db, to show stats on the website
func insertionUser(UserID string, serverID string) {
	stm, _ := db.Prepare("INSERT INTO inceneriti (UserID, TimeStamp, serverId) VALUES (?, ?, ?)")

	_, err := stm.Exec(UserID, time.Now(), serverID)
	if err != nil {
		lit.Error("Error inserting into the db, %s", err)
	}

	_ = stm.Close()
}

// Send a message in the configured text channel for the guild
func sendMessage(s *discordgo.Session, userID, guildID string) {
	var (
		n       int
		message string
		name    string
	)

	row := db.QueryRow("SELECT Name FROM utenti WHERE UserID = ?", userID)
	err := row.Scan(&name)
	if err != nil {
		lit.Error("Error scanning rows from query, %s", err)
		return
	}

	row = db.QueryRow("SELECT COUNT(*) FROM inceneriti WHERE UserID=? AND serverId=?", userID, guildID)
	err = row.Scan(&n)
	if err != nil {
		lit.Error("Error scanning rows from query, %s", err)
		return
	}

	// Otherwise Daniele "rompe il cazzo" for that final vowel if the number is 1
	if n == 1 {
		message = name + " è stato incenerito.\nÈ stato incenerito " + strconv.Itoa(n) + " volta."
	} else {
		message = name + " è stato incenerito.\nÈ stato incenerito " + strconv.Itoa(n) + " volte."
	}

	_, err = s.ChannelMessageSend(config[guildID].testuale, message)
	if err != nil {
		lit.Error("Error sending message, %s", err)
	}
}

// Saves roles of a user
func saveRoles(m *discordgo.Member, guildID string) {
	var roles string

	for _, r := range m.Roles {
		roles += r + ","
	}

	// User
	stm, _ := db.Prepare("INSERT IGNORE INTO utenti (UserID, Name) VALUES (?, ?)")

	_, err := stm.Exec(m.User.ID, m.User.Username)
	if err != nil {
		lit.Error("Error inserting into the db, %s", err)
	}

	_ = stm.Close()

	// Role
	stm, _ = db.Prepare("INSERT INTO roles (UserID, server, Roles, Nickname) VALUES (?, ?, ?, ?)")

	_, err = stm.Exec(m.User.ID, guildID, strings.TrimSuffix(roles, ","), m.Nick)
	if err != nil {
		lit.Error("Error inserting into the db, %s", err)
	}

	_ = stm.Close()
}

// Removes a given role for a given user
func removeRole(userID, guildID string) {
	stm, _ := db.Prepare("DELETE FROM roles WHERE UserID=? AND server=?")

	_, err := stm.Exec(userID, guildID)
	if err != nil {
		lit.Error("Error removing from the db, %s", err)
	}

	_ = stm.Close()
}

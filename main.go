package main

import (
	"database/sql"
	"github.com/bwmarrin/lit"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kkyr/fig"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	Token    string `fig:"token" validate:"required"`
	DSN      string `fig:"datasourcename" validate:"required"`
	Driver   string `fig:"drivername" validate:"required"`
	LogLevel string `fig:"loglevel" validate:"required"`
	Site     string `fig:"site" validate:"required"`
}

var (
	// Discord bot token
	token string
	// Config for all of the servers
	config = make(map[string]Server)
	// Database connection
	db *sql.DB
	// Stores if a userID is a bot or not
	isBot = make(map[string]*bool)
	// Playing status
	site string
)

func init() {
	lit.LogLevel = lit.LogError

	var cfg Config
	err := fig.Load(&cfg, fig.File("config.yml"))
	if err != nil {
		lit.Error(err.Error())
		return
	}

	// Config file found
	token = cfg.Token
	site = cfg.Site

	// Open database
	db, err = sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		lit.Error("Error opening database connection, %s", err)
		return
	}

	// Set lit.LogLevel to the given value
	switch strings.ToLower(cfg.LogLevel) {
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
	execQuery(tblUtenti)
	execQuery(tblConfig)
	execQuery(tblInceneriti)
	execQuery(tblRoles)

	// And loads the config for all the servers
	loadConfig()
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		lit.Error("Error creating Discord session, %s", err)
		return
	}

	// Add events handler
	dg.AddHandler(ready)
	dg.AddHandler(voiceStateUpdate)
	dg.AddHandler(guildMemberAdd)

	// Add commands handler
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	// Initialize intents that we use
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMembers | discordgo.IntentsGuildVoiceStates)

	// Open discord session
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
	err := s.UpdateGameStatus(0, site)
	if err != nil {
		lit.Error("Can't set status, %s", err)
	}

	// Checks for unused commands and deletes them
	if cmds, err := s.ApplicationCommands(s.State.User.ID, ""); err == nil {
		found := false

		for _, l := range commands {
			found = false

			for _, o := range cmds {
				// We compare every online command with the ones locally stored, to find if a command with the same name exists
				if l.Name == o.Name {
					// If the options of the command are not equal, we re-register it
					if !isCommandEqual(l, o) {
						lit.Info("Registering command `%s`", l.Name)

						_, err = s.ApplicationCommandCreate(s.State.User.ID, "", l)
						if err != nil {
							lit.Error("Cannot create '%s' command: %s", l.Name, err)
						}
					}

					found = true
					break
				}
			}

			// If we didn't found a match for the locally stored command, it means the command is new. We register it
			if !found {
				lit.Info("Registering new command `%s`", l.Name)

				_, err = s.ApplicationCommandCreate(s.State.User.ID, "", l)
				if err != nil {
					lit.Error("Cannot create '%s' command: %s", l.Name, err)
				}
			}
		}
	}
}

// Called when someone changes channel or enters one
func voiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	// Checks if the voice state update is from the correct channel and the user isn't a bot
	if isBot[v.UserID] == nil {
		user, err := s.User(v.UserID)
		if err == nil {
			isBot[v.UserID] = &user.Bot
		} else {
			lit.Error("User failed: %s", err.Error())
			return
		}
	}

	if *isBot[v.UserID] || v.ChannelID != config[v.GuildID].vocale {
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
	_, err = s.GuildMemberEdit(v.GuildID, v.UserID, &discordgo.GuildMemberParams{Roles: &[]string{config[v.GuildID].ruolo}})
	if err != nil {
		lit.Error("Error adding role, %s", err)

		addRoles(s, v.UserID, v.GuildID)
		return
	}

	// Search for the user private message channel
	canale, err := s.UserChannelCreate(v.UserID)
	if err != nil {
		lit.Error("Error getting DM channel id, %s", err)

		addRoles(s, v.UserID, v.GuildID)
		return
	}

	// Check if custom message to send exists
	if config[v.GuildID].messagge != "" {
		_, err = s.ChannelMessageSend(canale.ID, config[v.GuildID].messagge)
		if err != nil {
			lit.Error("Error sending message, %s", err)
		}
	}

	// Wait 3 seconds
	time.Sleep(3 * time.Second)

	// Send the invite link
	_, err = s.ChannelMessageSend(canale.ID, config[v.GuildID].invito)
	if err != nil {
		lit.Error("Error sending message, %s", err)
	}

	// And kicks the user
	err = s.GuildMemberDelete(v.GuildID, v.UserID)
	if err != nil {
		lit.Error("Error kicking user, %s", err)

		addRoles(s, v.UserID, v.GuildID)
		return
	}

	// Tracks when the user was kicked, to show on the website
	insertionUser(v.UserID, v.GuildID)
	// And sends the message on the guild text channel
	sendMessage(s, v.UserID, v.GuildID)
}

// Used to add roles&nick back to the user
func guildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	addRoles(s, m.User.ID, m.GuildID)
}

// Adds the user to the db, to show stats on the website
func insertionUser(UserID string, serverID string) {
	stm, _ := db.Prepare("INSERT INTO inceneriti (UserID, TimeStamp, serverId) VALUES (?, NOW(), ?)")

	_, err := stm.Exec(UserID, serverID)
	if err != nil {
		lit.Error("Error inserting into the db, %s", err)
	}

	_ = stm.Close()
}

// Send a message in the configured text channel for the guild
func sendMessage(s *discordgo.Session, userID, guildID string) {
	var (
		message, name string
		n             int
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

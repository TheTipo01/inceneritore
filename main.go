package main

import (
	"database/sql"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/lit"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kkyr/fig"
	cmap "github.com/orcaman/concurrent-map/v2"

	"github.com/bwmarrin/discordgo"
)

var (
	// Discord bot token
	token string
	// Config for all the servers
	config cmap.ConcurrentMap[string, Server]
	// Database connection
	db *sql.DB
	// Stores if a userID is a bot or not
	isBot *cmap.ConcurrentMap[string, bool]
	// Playing status
	site string
	// Stores if a user is being incinerated
	isUserBeingIncinerated = cmap.New[bool]()
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

	db.SetConnMaxLifetime(time.Minute * 3)

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
	execQuery(tblUsers, tblServers, tblRoles, tblIncinerated)

	config = cmap.New[Server]()
	for _, s := range cfg.Server {
		config.Set(s.ServerID, s)

		err = saveServer(s.ServerID, s.ServerName)
		if err != nil {
			lit.Error("Error saving server %s, %s", s.ServerID, err)
		}
	}

	isBot = loadIsBot()
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

	// Register commands
	_, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", commands)
	if err != nil {
		lit.Error("Can't register commands, %s", err)
	}

	// Wait here until CTRL-C or another term signal is received.
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
}

// Called when someone changes channel or enters one
func voiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	start := time.After(time.Second * 3)

	// Checks if the voice state update is from the correct channel and the user isn't a bot
	if !isBot.Has(v.UserID) {
		user, err := s.User(v.UserID)
		if err == nil {
			isBot.Set(v.UserID, user.Bot)
			err = saveIsBot(v.UserID, user.Username, user.Bot)
			if err != nil {
				lit.Error("Error saving user as bot, %s", err)
			}
		} else {
			lit.Error("User failed: %s", err.Error())
			return
		}
	}

	b, _ := isBot.Get(v.UserID)
	c, _ := config.Get(v.GuildID)

	if b || v.ChannelID != c.VoiceChannel {
		return
	}

	// Is the user already being incinerated?
	if isUserBeingIncinerated.Has(v.UserID) {
		return
	} else {
		isUserBeingIncinerated.Set(v.UserID, true)
		defer isUserBeingIncinerated.Remove(v.UserID)
	}

	m, err := s.GuildMember(v.GuildID, v.UserID)
	if err != nil {
		lit.Error("Error creating member, %s", err)
	}

	err = saveRoles(m, v.GuildID)
	if err != nil {
		lit.Error("Error saving roles, %s", err)
	}

	// We can't remove the role from a booster user, so we leave it there
	var guildMemberParams discordgo.GuildMemberParams
	if m.PremiumSince == nil {
		guildMemberParams.Roles = &[]string{c.Role}
	} else {
		guildMemberParams.Roles = &[]string{c.Role, c.BoostRole}
	}

	// Add the role, so the user doesn't move
	_, err = s.GuildMemberEdit(v.GuildID, v.UserID, &guildMemberParams)
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

	// Check if a custom message to send exists
	if c.Message != "" {
		_, err = s.ChannelMessageSend(canale.ID, c.Message)
		if err != nil {
			lit.Error("Error sending message, %s", err)
		}
	}

	// Wait 3 seconds since the start of the function
	<-start

	// Send the invite link
	_, err = s.ChannelMessageSend(canale.ID, c.Invite)
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
	insertIncenerimento(v.UserID, v.GuildID)
	// And sends the message on the guild text channel
	sendMessage(s, v.UserID, v.GuildID, m.User.Username, c.TextChannel)
}

// Used to add roles&nick back to the user
func guildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	addRoles(s, m.User.ID, m.GuildID)
}

// Send a message in the configured text channel for the guild
func sendMessage(s *discordgo.Session, userID, guildID, name, textChannel string) {
	var (
		message string
		n       int
	)

	n = getIncenerimenti(userID, guildID)

	// Otherwise, Daniele "rompe il cazzo" for that final vowel if the number is 1
	message = name + " è stato incenerito.\nÈ stato incenerito " + strconv.Itoa(n)
	if n == 1 {
		message += " volta."
	} else {
		message += " volte."
	}

	_, err := s.ChannelMessageSend(textChannel, message)
	if err != nil {
		lit.Error("Error sending message, %s", err)
	}
}

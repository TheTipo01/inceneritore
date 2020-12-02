package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"log"
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
	Token          string
	config         = make(map[string]Config)
	lastKick       = make(map[string]map[string]*time.Time)
	DataSourceName string
	driverName     = "mysql"
	database       *sql.DB
)

func init() {
	var err error

	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			fmt.Println("Config file not found! See example_config.yml")
			return
		}
	} else {
		// Config file found
		Token = viper.GetString("token")
		DataSourceName = viper.GetString("datasourcename")
	}

	database, err = sql.Open(driverName, DataSourceName)
	if err != nil {
		log.Println("Error opening DB,", err)
		return
	}

	execQuery(tblInceneriti, database)
	execQuery(tblRoles, database)
	execQuery(tblUtenti, database)
	execQuery(tblConfig, database)

	load(database)

}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(voiceStateUpdate)
	dg.AddHandler(messageCreate)
	dg.AddHandler(guildMemberAdd)
	dg.AddHandler(ready)

	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMembers | discordgo.IntentsGuildVoiceStates | discordgo.IntentsGuildMessages)

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	err = dg.Close()
	if err != nil {
		log.Println("Error closing session,", err)
	}

	err = database.Close()
	if err != nil {
		log.Println("Error closing database,", err)
	}
}

func ready(s *discordgo.Session, _ *discordgo.Ready) {

	// Set the playing status.
	err := s.UpdateStatus(0, "inceneritore.ga")
	if err != nil {
		fmt.Println("Can't set status,", err)
	}
}

// Carica i config per i server
func load(db *sql.DB) {
	var (
		serverID string
		ruolo    string
		testuale string
		vocale   string
		invito   string
		nome     string
	)

	rows, err := db.Query("SELECT * FROM config")
	if err != nil {
		log.Println("Error querying database,", err)
	}

	for rows.Next() {
		err = rows.Scan(&serverID, &nome, &ruolo, &testuale, &vocale, &invito)
		if err != nil {
			log.Println("Error scanning rows from query,", err)
			continue
		}

		if lastKick[serverID] == nil {
			lastKick[serverID] = make(map[string]*time.Time)
		}

		config[serverID] = Config{
			ruolo:    ruolo,
			testuale: testuale,
			vocale:   vocale,
			invito:   invito,
			nome:     nome,
		}
	}
}

// Chiamata quando qualcuno entra o viene spostato in un canale vocale
func voiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {

	if user, err := s.User(v.UserID); err == nil && (v.ChannelID != config[v.GuildID].vocale || user.Bot) {
		return
	}

	if lastKick[v.GuildID][v.UserID] != nil && time.Now().Sub(*lastKick[v.GuildID][v.UserID]) < time.Second {
		log.Println("Event fired twice")
		return
	}

	currentTime := time.Now()
	lastKick[v.GuildID][v.UserID] = &currentTime

	// Creo il membro da passare alla funzione per salvare ruoli
	m, err := s.GuildMember(v.GuildID, v.UserID)
	if err != nil {
		log.Println("Error creating member,", err)
	}

	saveRoles(m, v.GuildID)

	// Aggiungo ruolo
	err = s.GuildMemberRoleAdd(v.GuildID, v.UserID, config[v.GuildID].ruolo)
	if err != nil {
		log.Println("Error adding role,", err)

		removeRole(m, v.GuildID)
		return
	}

	// Aspetto 3 secondi
	time.Sleep(3 * time.Second)

	// Cerco id del canale per inviare DM
	canale, err := s.UserChannelCreate(v.UserID)
	if err != nil {
		log.Println("Error getting DM channel id,", err)

		_ = s.GuildMemberRoleRemove(v.GuildID, v.UserID, config[v.GuildID].ruolo)

		removeRole(m, v.GuildID)
		return
	}

	// Invio messaggio con l'invito
	_, err = s.ChannelMessageSend(canale.ID, config[v.GuildID].invito)
	if err != nil {
		log.Println("Error sending message,", err)
	}

	// Espello l'utente
	err = s.GuildMemberDelete(v.GuildID, v.UserID)
	if err != nil {
		log.Println("Error kicking user,", err)

		//Se non riesco tolgo il ruolo
		_ = s.GuildMemberRoleRemove(v.GuildID, v.UserID, config[v.GuildID].ruolo)

		removeRole(m, v.GuildID)
		return
	}

	// Chiamata per le funzioni per inserimento nel db ed invio messaggio nel canale cestino
	insertionUser(v.UserID, v.GuildID)
	sendMessage(s, v)

}

// Chiamata quando un messaggio viene creato, usato per rispondere al comando ?inceneriti
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Content != "?inceneriti" || m.Author.Bot || m.Author.ID == s.State.User.ID {
		return
	}

	var (
		name string
		mex  string
		cont int
	)

	// Querying database
	rows, err := database.Query("SELECT Name, Count(inceneriti.UserID) FROM inceneriti, utenti WHERE utenti.UserID = inceneriti.UserID AND serverID=? GROUP BY inceneriti.UserID ORDER BY Count(inceneriti.UserID) DESC", m.GuildID)
	if err != nil {
		log.Println("Error querying database,", err)
	}

	for rows.Next() {
		err = rows.Scan(&name, &cont)
		if err != nil {
			log.Println("Error scanning rows from query,", err)
		}

		mex += name + " - " + strconv.Itoa(cont) + "\n\n"
	}

	_, err = s.ChannelMessageSend(m.ChannelID, mex)
	if err != nil {
		log.Println("Error sending message,", err)
	}
}

// Chiamata quando un utente entra nel server, per ridargli i ruoli
func guildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	var roles, nickname string

	rows, err := database.Query("SELECT Roles, Nickname FROM roles WHERE UserID=? AND server=?", m.User.ID, m.GuildID)
	if err != nil {
		log.Println("Error preparing query,", err)
	}

	for rows.Next() {
		err = rows.Scan(&roles, &nickname)
		if err != nil {
			log.Println("Error scanning rows from query,", err)
		}
	}

	statement, err := database.Prepare("DELETE FROM roles WHERE UserID=? AND server=?")
	if err != nil {
		log.Println("Error preparing insertion,", err)
	}

	_, err = statement.Exec(m.User.ID, m.GuildID)
	if err != nil {
		log.Println("Error deleting from database,", err)
	}

	err = s.GuildMemberNickname(m.GuildID, m.User.ID, nickname)
	if err != nil {
		log.Println("Error changing nickname,", err)
	}

	for _, role := range strings.Split(roles, ",") {
		if role != config[m.GuildID].ruolo && role != "" {
			err = s.GuildMemberRoleAdd(m.GuildID, m.User.ID, role)
			if err != nil {
				log.Println("Error adding role,", err)
			}
		}
	}

}

// Provvede all'inserimento dell'incenerito nel DB
func insertionUser(UserID string, serverID string) {

	statement, err := database.Prepare("INSERT INTO inceneriti (UserID, TimeStamp, serverId) VALUES (?, ?, ?)")
	if err != nil {
		log.Println("Error preparing query,", err)
	}

	_, err = statement.Exec(UserID, time.Now(), serverID)
	if err != nil {
		log.Println("Error inserting into the database,", err)
	}

}

// Provvede all'invio del messaggio nel canale #cestino
func sendMessage(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	var (
		numero    int
		messaggio string
		name      string
	)

	row := database.QueryRow("SELECT Name FROM utenti WHERE UserID = ?", v.UserID)
	err := row.Scan(&name)
	if err != nil {
		log.Println("Error scanning rows from query,", err)
		return
	}

	row = database.QueryRow("SELECT COUNT(*) FROM inceneriti WHERE UserID=? AND serverId=?", v.UserID, v.GuildID)
	err = row.Scan(&numero)
	if err != nil {
		log.Println("Error scanning rows from query,", err)
		return
	}

	// Altrimenti Daniele rompe il cazzo per quella vocale alla fine se il numero è uguale a 1
	if numero == 1 {
		messaggio = name + " è stato incenerito.\nÈ stato incenerito " + strconv.Itoa(numero) + " volta."
	} else {
		messaggio = name + " è stato incenerito.\nÈ stato incenerito " + strconv.Itoa(numero) + " volte."
	}

	_, err = s.ChannelMessageSend(config[v.GuildID].testuale, messaggio)
	if err != nil {
		log.Println("Error sending message,", err)
	}
}

// Salva i ruoli di un persona prima dell'incenerimento
func saveRoles(m *discordgo.Member, guildID string) {

	var roles string

	for _, r := range m.Roles {
		roles += r + ","
	}

	// Utente
	statement, err := database.Prepare("INSERT INTO utenti (UserID, Name) VALUES (?, ?)")
	if err != nil {
		log.Println("Error preparing insertion,", err)
	}

	_, err = statement.Exec(m.User.ID, m.User.Username)
	if err != nil {
		log.Println("Error inserting into the database,", err)
	}

	// Ruolo
	statement, err = database.Prepare("INSERT INTO roles (UserID, server, Roles, Nickname) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Println("Error preparing insertion,", err)
	}

	_, err = statement.Exec(m.User.ID, guildID, strings.TrimSuffix(roles, ","), m.Nick)
	if err != nil {
		log.Println("Error inserting into the database,", err)
	}

}

// Rimuove il ruolo inserito nel DB, per evitare di sporcarlo di tuple inutili
func removeRole(m *discordgo.Member, guildID string) {
	statement, err := database.Prepare("DELETE FROM roles WHERE UserID=? AND server=?")
	if err != nil {
		log.Println("Error preparing insertion,", err)
	}

	_, err = statement.Exec(m.User.ID, guildID)
	if err != nil {
		log.Println("Error removing from the database,", err)
	}
}

package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
)

var (
	// Commands
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "inceneriti",
			Description: "Prints the ranking of the most incinerated people",
		},
		{
			Name:        "update",
			Description: "Updates the username of the user: this is useful if you changed your username.",
		},
	}

	// Handler
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		// Prints the ranking of the most incinerated people
		"inceneriti": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var (
				name, mex   string
				times, cont int
			)

			// Querying db
			rows, err := db.Query("SELECT Name, Count(inceneriti.UserID) FROM inceneriti, utenti WHERE utenti.UserID = inceneriti.UserID AND serverID=? GROUP BY inceneriti.UserID ORDER BY Count(inceneriti.UserID) DESC", i.GuildID)
			if err != nil {
				lit.Error("Error querying db, %s", err)
			}

			for rows.Next() {
				cont++
				err = rows.Scan(&name, &times)
				if err != nil {
					lit.Error("Error scanning rows from query, %s", err)
					continue
				}

				mex += fmt.Sprintf("%d) %s - %d\n", cont, name, times)
			}

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Ranking", mex).
				SetColor(0x7289DA).MessageEmbed, i.Interaction)
		},
		// Updates the username of the user: this is useful if you changed your username.
		"update": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Querying db
			_, _ = db.Query("UPDATE utenti SET Name=? WHERE UserID=?", i.Member.User.Username, i.Member.User.ID)

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).SetDescription("Updated username!").
				SetColor(0x7289DA).MessageEmbed, i.Interaction)
		},
	}
)

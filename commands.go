package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	"strconv"
)

var (
	// Commands
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "inceneriti",
			Description: "Prints the ranking of the most incinerated people",
		},
	}

	// Handler
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		// Prints the ranking of the most incinerated people
		"inceneriti": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var (
				name, mex string
				cont      int
			)

			// Querying db
			rows, err := db.Query("SELECT Name, Count(inceneriti.UserID) FROM inceneriti, utenti WHERE utenti.UserID = inceneriti.UserID AND serverID=? GROUP BY inceneriti.UserID ORDER BY Count(inceneriti.UserID) DESC", i.GuildID)
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

			sendEmbedInteraction(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Ranking", mex).
				SetColor(0x7289DA).MessageEmbed, i.Interaction)
		},
	}
)

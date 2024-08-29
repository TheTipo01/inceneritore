package main

import (
	"database/sql"
	"errors"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	"strings"
)

// Sends embed as response to an interaction
func sendEmbedInteraction(s *discordgo.Session, embed *discordgo.MessageEmbed, i *discordgo.Interaction) {
	sliceEmbed := []*discordgo.MessageEmbed{embed}
	err := s.InteractionRespond(i, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: sliceEmbed}})
	if err != nil {
		lit.Error("InteractionRespond failed: %s", err)
		return
	}
}

// Gives back roles after a person has been incinerated
func addRoles(s *discordgo.Session, userID, guildID string) {
	var roles, nickname string

	err := db.QueryRow("SELECT Roles, Nickname FROM roles WHERE UserID=? AND server=?", userID, guildID).Scan(&roles, &nickname)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			// Nothing to do
			return
		}
		lit.Error("Error scanning row from query, %s", err)
	}

	_, err = db.Exec("DELETE FROM roles WHERE UserID=? AND server=?", userID, guildID)
	if err != nil {
		lit.Error("Error deleting from db, %s", err)
	}

	// Change nickname, if any
	if nickname != "" {
		err = s.GuildMemberNickname(guildID, userID, nickname)
		if err != nil {
			lit.Error("Error changing nickname, %s", err)
		}
	}

	var splittedRoles []string
	if roles != "" {
		splittedRoles = strings.Split(roles, ",")
	}

	// Remove the incenerito role if the user has it
	c, _ := config.Get(guildID)
	for i, v := range splittedRoles {
		if v == c.ruolo {
			splittedRoles = append(splittedRoles[:i], splittedRoles[i+1:]...)
			break
		}
	}

	// Add the roles if the user has any
	if len(splittedRoles) > 0 {
		_, err = s.GuildMemberEdit(guildID, userID, &discordgo.GuildMemberParams{Roles: &splittedRoles})
		if err != nil {
			lit.Error("Error adding role, %s", err)
		}
	}
}

package main

import (
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

// Gives back role after a person has been incinerated
func addRoles(s *discordgo.Session, userID, guildID string) {
	var roles, nickname string

	err := db.QueryRow("SELECT Roles, Nickname FROM roles WHERE UserID=? AND server=?", userID, guildID).Scan(&roles, &nickname)
	if err != nil {
		lit.Error("Error scanning row from query, %s", err)
	}

	stm, _ := db.Prepare("DELETE FROM roles WHERE UserID=? AND server=?")

	_, err = stm.Exec(userID, guildID)
	if err != nil {
		lit.Error("Error deleting from db, %s", err)
	}

	_ = stm.Close()

	err = s.GuildMemberNickname(guildID, userID, nickname)
	if err != nil {
		lit.Error("Error changing nickname, %s", err)
	}

	err = s.GuildMemberEdit(guildID, userID, strings.Split(roles, ","))
	if err != nil {
		lit.Error("Error adding role, %s", err)
	}
}

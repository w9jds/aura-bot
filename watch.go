package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	esi "github.com/w9jds/go.esi"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func postSystemWatchMail(channelId string, mail *esi.KillMail, names map[uint]esi.NameRef, value float64) {
	var title string
	printer := message.NewPrinter(language.English)

	killer := killer(mail)
	victim := CharacterRef{
		CharacterID: uint(mail.Victim.ID),
		AllianceID:  uint(mail.Victim.AllianceID),
		CorpID:      uint(mail.Victim.CorporationID),
	}

	victim.getLinks(names)
	killer.getLinks(names)

	title = fmt.Sprintf("%s killed %s (%s)", names[uint(killer.CharacterID)].Name, names[uint(mail.Victim.ID)].Name, names[uint(mail.Victim.CorporationID)].Name)

	if len(title) > 256 {
		title = title[:253] + "..."
	}

	message := &discordgo.MessageSend{
		Embed: &discordgo.MessageEmbed{
			Title:     title,
			URL:       fmt.Sprintf("https://zkillboard.com/kill/%d", mail.ID),
			Timestamp: mail.Time,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: fmt.Sprintf("https://imageserver.eveonline.com/Render/%d_128.png", mail.Victim.ShipTypeID),
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Killer",
					Value:  killer.Display,
					Inline: true,
				},
				{
					Name:   "Victim",
					Value:  victim.Display,
					Inline: true,
				},
				{
					Name:   "Ship",
					Value:  fmt.Sprintf("[%s](https://zkillboard.com/ship/%d)", names[uint(mail.Victim.ShipTypeID)].Name, mail.Victim.ShipTypeID),
					Inline: false,
				},
				{
					Name:   "System",
					Value:  fmt.Sprintf("[%s](https://zkillboard.com/system/%d)", names[uint(mail.SystemID)].Name, mail.SystemID),
					Inline: false,
				},
				{
					Name:   "Pilots Involved",
					Value:  fmt.Sprintf("%d", len(mail.Attackers)),
					Inline: true,
				},
				{
					Name:   "Value",
					Value:  printer.Sprintf("%.2f ISK", value),
					Inline: true,
				},
			},
		},
	}

	_, err := aura.ChannelMessageSendComplex(channelId, message)
	if err != nil {
		log.Println(err)
	}
}

func killer(mail *esi.KillMail) CharacterRef {
	var finalBlow CharacterRef

	for _, attacker := range mail.Attackers {
		if attacker.FinalBlow {
			finalBlow = CharacterRef{
				CharacterID: uint(attacker.ID),
				CorpID:      uint(attacker.CorporationID),
				AllianceID:  uint(attacker.AllianceID),
			}
		}
	}

	return finalBlow
}

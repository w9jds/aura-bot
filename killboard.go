package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	esi "github.com/w9jds/go.esi"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type KillBoard struct {
	GroupID   uint   `json:"group_id"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
}

type CharacterRef struct {
	CharacterID uint
	AllianceID  uint
	CorpID      uint

	Display string
}

func (ref CharacterRef) getAffiliation() uint {
	var affiliation uint

	if ref.CorpID != 0 {
		affiliation = ref.CorpID
	}

	if ref.AllianceID != 0 {
		affiliation = ref.AllianceID
	}

	return affiliation
}

func (ref *CharacterRef) getLinks(names map[uint]esi.NameRef) {
	var affiliation string
	characterLink := fmt.Sprintf("[%s](https://zkillboard.com/character/%d)", names[ref.CharacterID].Name, ref.CharacterID)

	if ref.CorpID != 0 {
		affiliation = fmt.Sprintf("[%s](https://zkillboard.com/corporation/%d)", names[ref.CorpID].Name, ref.CorpID)
	}

	if ref.AllianceID != 0 {
		affiliation = fmt.Sprintf("[%s](https://zkillboard.com/alliance/%d)", names[ref.AllianceID].Name, ref.AllianceID)
	}

	ref.Display = fmt.Sprintf("%s\n%s", characterLink, affiliation)
}

func (storage Storage) CreateKillBoard(channelID string, guildID string, option *discordgo.ApplicationCommandInteractionDataOption) {
	killboard := `INSERT INTO channels (channel_id, guild_id, type, group_id) VALUES ($1, $2, 'KillBoard', $3)`

	_, err := storage.database.Exec(killboard, channelID, guildID, option.UintValue())
	if err != nil {
		log.Println(err)
	}
}

func (storage Storage) FindKillboard(channelID string, guildID string) (uint, error) {
	var groupID uint
	query := `SELECT group_ID FROM channels WHERE type = 'KillBoard' AND channel_id=$1 AND guild_id=$2`

	row := storage.database.QueryRow(query, channelID, guildID)
	switch error := row.Scan(&groupID); error {
	case sql.ErrNoRows:
		return groupID, sql.ErrNoRows
	case nil:
		return groupID, nil
	default:
		return groupID, error
	}
}

func (storage Storage) FindKillboards(ids []uint) []KillBoard {
	killboards := []KillBoard{}
	conv := []string{}

	for _, id := range ids {
		conv = append(conv, fmt.Sprintf("%d", id))
	}

	values := strings.Join(conv[:], ",")
	query := fmt.Sprintf(`SELECT channel_id, guild_id, group_id FROM channels WHERE type = 'KillBoard' AND group_id IN (%s)`, values)

	rows, err := storage.database.Query(query)
	if err != nil {
		log.Println(err)
	} else {
		for rows.Next() {
			board, err := readKillBoard(rows)
			if err != nil {
				log.Println(err)
			} else {
				killboards = append(killboards, board)
			}
		}
	}

	return killboards
}

func shareKillBoardMail(board KillBoard, mail *esi.KillMail, names map[uint]esi.NameRef, value float64) {
	var title string
	printer := message.NewPrinter(language.English)

	isVictim := victim(board, mail)
	isAttacker, killer, friendlies := attackers(board, mail)

	victim := CharacterRef{
		CharacterID: uint(mail.Victim.ID),
		AllianceID:  uint(mail.Victim.AllianceID),
		CorpID:      uint(mail.Victim.CorporationID),
	}

	victim.getLinks(names)
	killer.getLinks(names)

	if !isVictim && !isAttacker {
		title = fmt.Sprintf("%s lost a %s to %s", names[uint(mail.Victim.CorporationID)].Name, names[uint(mail.Victim.ShipTypeID)].Name, names[killer.getAffiliation()].Name)
	} else {
		title = fmt.Sprintf("%s killed %s (%s)", names[uint(killer.CharacterID)].Name, names[uint(mail.Victim.ID)].Name, names[uint(mail.Victim.CorporationID)].Name)
	}

	if len(title) > 256 {
		title = title[:253] + "..."
	}

	message := &discordgo.MessageSend{
		Embed: &discordgo.MessageEmbed{
			Title:     title,
			URL:       fmt.Sprintf("https://zkillboard.com/kill/%d", mail.ID),
			Timestamp: mail.Time,
			Color:     6710886,
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

	if isAttacker {
		message.Embed.Color = 8103679
	}

	if isVictim {
		message.Embed.Color = 16711680
	}

	if isVictim && isAttacker {
		message.Embed.Color = 6570404
	}

	if len(friendlies) > 0 {
		var members []string
		var participants string

		for _, id := range friendlies {
			members = append(members, fmt.Sprintf("[%s](https://zkillboard.com/character/%d/)", names[uint(id)].Name, id))
		}

		for {
			participants = strings.Join(members, ", ")

			if len(participants) <= 1024 {
				break
			}

			members = members[:len(members)-1]
		}

		message.Embed.Fields = append(message.Embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Friendly Pilots Involved",
			Value:  participants,
			Inline: false,
		})
	}

	_, err := aura.ChannelMessageSendComplex(board.ChannelID, message)
	if err != nil {
		log.Println(err)
	}
}

func victim(board KillBoard, mail *esi.KillMail) bool {
	return mail.Victim.AllianceID == uint32(board.GroupID) || mail.Victim.CorporationID == uint32(board.GroupID)
}

func attackers(board KillBoard, mail *esi.KillMail) (bool, CharacterRef, []uint32) {
	var finalBlow CharacterRef
	friendlies := []uint32{}
	isAttacker := false

	for _, attacker := range mail.Attackers {
		isFriendly := attacker.CorporationID == uint32(board.GroupID) || attacker.AllianceID == uint32(board.GroupID)

		if isFriendly {
			isAttacker = true
			friendlies = append(friendlies, attacker.ID)
		}

		if attacker.FinalBlow {
			finalBlow = CharacterRef{
				CharacterID: uint(attacker.ID),
				CorpID:      uint(attacker.CorporationID),
				AllianceID:  uint(attacker.AllianceID),
			}
		}
	}

	return isAttacker, finalBlow, friendlies
}

func readKillBoard(row *sql.Rows) (KillBoard, error) {
	board := KillBoard{}

	switch error := row.Scan(&board.ChannelID, &board.GuildID, &board.GroupID); error {
	case nil:
		return board, nil
	default:
		return board, error
	}
}

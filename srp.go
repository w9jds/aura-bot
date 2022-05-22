package main

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	esi "github.com/w9jds/go.esi"
	evepraisal "github.com/w9jds/go.evepraisal"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func findKillID(contents string) string {
	words := strings.Split(contents, " ")

	for _, word := range words {
		if strings.HasPrefix(strings.ToLower(word), "https://zkillboard.com/") {
			uri, error := url.Parse(word)
			if error != nil {
				return ""
			}

			params := strings.Split(uri.Path, "/")
			for _, param := range params {
				if _, error := strconv.Atoi(param); error == nil {
					return param
				}
			}

			return ""
		}
	}

	return ""
}

func formatCurrency(value float64) string {
	printer := message.NewPrinter(language.English)
	return printer.Sprintf("%.2f ISK", value)
}

func getHullAppraisal(killmail *esi.KillMail, fitting *esi.KillFitting) (*evepraisal.Response, error) {
	items := []*evepraisal.AppraisalItem{
		{
			TypeID:   killmail.Victim.ShipTypeID,
			Quantity: 1,
		},
	}

	if len(fitting.SubSystemSlot) > 0 {
		for id := range fitting.SubSystemSlot {
			items = append(items, &evepraisal.AppraisalItem{
				TypeID:   id,
				Quantity: 1,
			})
		}
	}

	return appraisalClient.AppraiseAll(items, "jita")
}

func addFittingItems(items []*evepraisal.AppraisalItem, slots map[uint32]*esi.KillItem) []*evepraisal.AppraisalItem {
	for id := range slots {
		items = append(items, &evepraisal.AppraisalItem{
			TypeID:   id,
			Quantity: slots[id].QuantityDestroyed + slots[id].QuantityDropped,
		})
	}

	return items
}

func getFittedAppraisal(killmail *esi.KillMail, fitting *esi.KillFitting) (*evepraisal.Response, error) {
	items := []*evepraisal.AppraisalItem{
		{
			TypeID:   killmail.Victim.ShipTypeID,
			Quantity: 1,
		},
	}

	items = addFittingItems(items, fitting.HighSlot)
	items = addFittingItems(items, fitting.MedSlot)
	items = addFittingItems(items, fitting.LoSlot)
	items = addFittingItems(items, fitting.RigSlot)

	return appraisalClient.AppraiseAll(items, "jita")
}

func srpRequest(session *discordgo.Session, message *discordgo.MessageCreate) {
	killID := findKillID(message.Content)

	metadata, error := zkbClient.GetKillMail(killID)
	if error != nil {
		log.Println(error)
		session.ChannelMessageSend(message.ChannelID, "There was a problem processing the killmail link you provided.")
		return
	}

	killmail, fitting, error := esiClient.GetKillMail(metadata.ID, metadata.Zkb.Hash, true)
	if error != nil {
		log.Println(error)
		session.ChannelMessageSend(message.ChannelID, "There was a problem contacting ZKill with the link you provided.")
		return
	}

	insurance, error := esiClient.GetShipInsurance(killmail.Victim.ShipTypeID)
	if error != nil {
		log.Println(error)
		session.ChannelMessageSend(message.ChannelID, "There was a problem getting your insurance rates. Maybe ESI is down?")
		return
	}

	names := storage.FindNames([]uint{uint(killmail.Victim.ShipTypeID), uint(killmail.Victim.ID)})
	hullAppraisal, error := getHullAppraisal(killmail, fitting)
	if error != nil {
		log.Panic(error)
		session.ChannelMessageSend(message.ChannelID, "There was a problem getting the appraisal for your ship hull. Maybe EvePraisal is down?")
		return
	}

	fittedAppraisal, error := getFittedAppraisal(killmail, fitting)
	if error != nil {
		log.Panic(error)
		session.ChannelMessageSend(message.ChannelID, "There was a problem getting the appraisal for your ship fittings. Maybe EvePraisal is down?")
		return
	}

	_, error = session.ChannelMessageSendEmbed(message.ChannelID, &discordgo.MessageEmbed{
		Title: fmt.Sprintf("SRP Request for %s", names[uint(killmail.Victim.ID)].Name),
		Color: 16711680,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: fmt.Sprintf("https://imageserver.eveonline.com/Render/%d_128.png", killmail.Victim.ShipTypeID),
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Request processed %s", time.Now().UTC().Format("Monday, 02 January, 2006 15:04:05")),
		},
		URL: fmt.Sprintf("https://zkillboard.com/kill/%s/", killID),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Pilot",
				Value:  names[uint(killmail.Victim.ID)].Name,
				Inline: true,
			},
			{
				Name:   "Ship Hull",
				Value:  names[uint(killmail.Victim.ShipTypeID)].Name,
				Inline: true,
			},
			{
				Name:   "Platinum Insurance Payout",
				Value:  formatCurrency(insurance.Platinum.Payout),
				Inline: false,
			},
			{
				Name:   "Hull Value (Jita Sell)",
				Value:  formatCurrency(hullAppraisal.Appraisal.Totals.Sell),
				Inline: true,
			},
			{
				Name:   "Hull Payout Amount",
				Value:  formatCurrency(hullAppraisal.Appraisal.Totals.Sell - insurance.Platinum.Payout),
				Inline: false,
			},
			{
				Name:   "Fitted Value (Jita Sell)",
				Value:  formatCurrency(fittedAppraisal.Appraisal.Totals.Sell),
				Inline: true,
			},
			{
				Name:   "Full Payout Amount",
				Value:  formatCurrency(fittedAppraisal.Appraisal.Totals.Sell - insurance.Platinum.Payout),
				Inline: false,
			},
		},
	})

	if error == nil {
		session.ChannelMessageDelete(message.ChannelID, message.ID)
	}
}

func srpPaidReaction(session *discordgo.Session, reactionAdd *discordgo.MessageReactionAdd) {
	message, error := session.ChannelMessage(reactionAdd.ChannelID, reactionAdd.MessageID)
	if error != nil {
		log.Print(error)
		return
	}

	if message.Author.Bot == false && message.Author.Username == "Aura" {
		return
	}

	if message.Embeds[0].Color != 38144 {
		message.Embeds[0].Color = 38144
		message.Embeds[0].Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Request paid %s", time.Now().UTC().Format("Monday, 02 January, 2006 15:04:05")),
		}
		session.ChannelMessageEditEmbed(reactionAdd.ChannelID, reactionAdd.MessageID, message.Embeds[0])
	}
}

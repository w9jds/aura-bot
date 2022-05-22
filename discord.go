package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "time",
			Description: "Get current EVE Time",
		},
		{
			Name:        "create",
			Description: "Register channel to display kills related to Alliance/Corporation",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "killboard",
					Description: "Create a Corp/Alliance Killboard",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "groupid",
							Description: "ID of corporation or alliance to show killmails for",
							Required:    true,
						},
					},
				},
				{
					Name:        "system-watch",
					Description: "Create a System Killboard",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "systemid",
							Description: "ID of solar system to show killmails from",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionNumber,
							Name:        "min-value",
							Description: "Value of all kills posted will be higher than",
							Required:    false,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "ignore-list",
							Description: "List of IDs that if the kill contains them, it's ignored (ie. `98709230,98636363,99010417`)",
							Required:    false,
						},
					},
				},
				// {
				// 	Name:        "srp",
				// 	Description: "Create a SRP Channel",
				// 	Type:        discordgo.ApplicationCommandOptionSubCommand,
				// 	Options:     []*discordgo.ApplicationCommandOption{},
				// },
			},
		},
		// {
		// 	Name:        "srp",
		// 	Description: "Log a new SRP request",
		// 	Options: []*discordgo.ApplicationCommandOption{
		// 		{
		// 			Type:        discordgo.ApplicationCommandOptionString,
		// 			Name:        "link",
		// 			Description: "Link to the zKill for your loss",
		// 			Required:    true,
		// 		},
		// 	},
		// },
	}

	handlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		// "srp":    handleSRP,
		"create": handleCreate,
		"time": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: time.Now().UTC().Format("Monday, 02 January, 2006 15:04:05"),
				},
			})
		},
	}
)

// func handleSRP(s *discordgo.Session, i *discordgo.InteractionCreate) {
// 	options := getCommandOptions(i.ApplicationCommandData().Options)
// 	log.Println(options["link"])
// }

func handleCreate(session *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	subcommand := getCommandOptions(options[0].Options)

	switch options[0].Name {
	case "killboard":
		groupId, err := storage.FindKillboard(i.ChannelID, i.GuildID)
		if err != nil {
			log.Println(err)
		}

		if groupId > 0 {
			session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("This channel is already registered as a Killboard for %d", groupId),
				},
			})
		} else {
			go storage.CreateKillBoard(i.ChannelID, i.GuildID, uint(subcommand["groupid"].UintValue()))
			session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("This channel will now display killmails with %d on them!", subcommand["groupid"].IntValue()),
				},
			})
		}
	case "system-watch":
		response := "This channel will now display killmails from: ```"
		var systemId uint
		minValue := 0.0
		ignoreList := ""

		if value, ok := subcommand["systemid"]; ok {
			systemId = uint(value.IntValue())
			response += fmt.Sprintf("\nSystem: %d", systemId)
		}

		if value, ok := subcommand["min-value"]; ok {
			minValue = value.FloatValue()
			printer := message.NewPrinter(language.English)
			response += fmt.Sprintf("\nMinimum value: %s", printer.Sprintf("%.2f ISK", minValue))
		}

		if value, ok := subcommand["ignore-list"]; ok {
			ignoreList = value.StringValue()
			response += fmt.Sprintf("\nIgnoring kills with: %s", ignoreList)
		}

		go storage.CreateSystemWatch(i.ChannelID, i.GuildID, uint(systemId), ignoreList, minValue)
		session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: response + "```",
			},
		})
	case "srp":

	}

}

func CreateAura() *discordgo.Session {
	token := getEnv("BOT_TOKEN", true)
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalln("Error creating discord client: ", err)
	}

	discord.AddHandlerOnce(ready)
	discord.AddHandler(serverAdded)
	discord.AddHandler(serverRemove)
	discord.AddHandler(channelDelete)
	discord.AddHandler(command)

	discord.Open()

	return discord
}

func registerCommands(session *discordgo.Session, guild *discordgo.Guild) {
	for _, command := range commands {
		_, err := session.ApplicationCommandCreate(session.State.User.ID, guild.ID, command)
		if err != nil {
			log.Printf("Cannot create '%v' command: %v", command.Name, err)
		}

		// registeredCommands[key] = cmd
	}
}

func shutdown(session *discordgo.Session) {
	for _, guild := range session.State.Guilds {
		for _, command := range commands {
			err := session.ApplicationCommandDelete(session.State.User.ID, guild.ID, command.ID)
			if err != nil {
				log.Printf("Cannot delete '%v' command: %v", command.Name, err)
			}
		}
	}
}

func ready(session *discordgo.Session, ready *discordgo.Ready) {
	log.Println("Aura has started! All systems green.")

	for _, guild := range session.State.Guilds {
		go registerCommands(session, guild)
	}
}

func channelDelete(session *discordgo.Session, channel *discordgo.ChannelDelete) {
	go storage.RemoveChannel(channel.ID, channel.GuildID)
}

func serverAdded(session *discordgo.Session, guild *discordgo.GuildCreate) {
	go registerCommands(session, guild.Guild)
}

func serverRemove(session *discordgo.Session, guild *discordgo.GuildDelete) {
	go storage.CleanupGuild(guild.ID)
}

func command(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	command := interaction.ApplicationCommandData()
	if handler, ok := handlers[command.Name]; ok {
		go handler(session, interaction)
	}
}

func getCommandOptions(options []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	return optionMap
}

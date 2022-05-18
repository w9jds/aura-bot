package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
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
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "groupid",
					Description: "ID of corporation or alliance to show killmails for",
					Required:    true,
				},
			},
		},
		{
			Name:        "srp",
			Description: "Log a new SRP request",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "link",
					Description: "Link to the zKill for your loss",
					Required:    true,
				},
			},
		},
	}

	handlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"srp":    handleSRP,
		"create": handleCreateFeed,
		"time": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		},
	}
)

func handleSRP(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := getCommandOptions(i)

	log.Println(options["link"])
}

func handleCreateFeed(session *discordgo.Session, i *discordgo.InteractionCreate) {
	options := getCommandOptions(i)

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
		go storage.CreateKillBoard(i.ChannelID, i.GuildID, uint(options["groupid"].UintValue()))
		session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("This channel will now display killmails with %d on them!", options["groupid"].IntValue()),
			},
		})
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

func ready(session *discordgo.Session, ready *discordgo.Ready) {
	log.Println("Aura has started! All systems green.")
}

func channelDelete(session *discordgo.Session, channel *discordgo.ChannelDelete) {
	go storage.RemoveChannel(channel.ID, channel.GuildID)
}

func serverAdded(session *discordgo.Session, guild *discordgo.GuildCreate) {
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))

	for key, command := range commands {
		cmd, err := session.ApplicationCommandCreate(session.State.User.ID, guild.ID, command)
		if err != nil {
			log.Printf("Cannot create '%v' command: %v", command.Name, err)
		}

		registeredCommands[key] = cmd
	}
}

func serverRemove(session *discordgo.Session, guild *discordgo.GuildDelete) {
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	go storage.CleanupGuild(guild.ID)

	for _, command := range registeredCommands {
		err := session.ApplicationCommandDelete(session.State.User.ID, guild.ID, command.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", command.Name, err)
		}
	}
}

func command(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	command := interaction.ApplicationCommandData()
	if handler, ok := handlers[command.Name]; ok {
		go handler(session, interaction)
	}
}

func getCommandOptions(i *discordgo.InteractionCreate) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	options := i.ApplicationCommandData().Options

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	return optionMap
}

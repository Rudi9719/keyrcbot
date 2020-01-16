package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rudi9719/loggy"
	"github.com/whyrusleeping/hellabot"
	"samhofi.us/x/keybase"
)

var (
	dev     = false
	k       = keybase.NewKeybase()
	channel keybase.Channel
	irc     *hbot.Bot
	botNick = "keyrcbot"                                                           // Set this for IRC Nickname
	serv    = flag.String("server", "chat.freenode.net:6667", "hostname and port") // Set this for whatever server you're using
	nick    = flag.String("nick", botNick, "nickname for the bot")
	logOpts = loggy.LogOpts{
		OutFile:   "irc.log",            // Set this for Logging output to file
		KBTeam:    "nightmarehaus.bots", // Set this for Logging output to Keybase team
		KBChann:   "general",            // Required for logging to Keybase Team, can be any channel
		ProgName:  "irclink",            // Also required for logging to Keybase team
		Level:     3,                    // 1 = Critical only, 2 = Errors, 3 = include Warnings, 4 = Debug, 5 = Info
		UseStdout: true,                 // Set to true to also display to stdout
	}
	log = loggy.NewLogger(logOpts)
)

func main() {
	log.LogWarn("Starting bot")
	if !k.LoggedIn {
		log.LogPanic("You are not logged in.")
	}
	channel.MembersType = keybase.TEAM
	channel.Name = "keyrc"        // The team you're linking to IRC
	channel.TopicName = "general" // The control channel (will be ignored for all except commands)
	go setupIRC()
	k.Run(func(api keybase.ChatAPI) {
		handleMessage(api)
	})
}

func setupIRC() {
	var err error
	saslOption := func(bot *hbot.Bot) {
		bot.SASL = true
		bot.Password = os.Getenv("IRC_PASS") // Set this to authenticate with IRC
	}
	hijackSession := func(bot *hbot.Bot) {
		bot.HijackSession = true
		log.LogWarn("setting irc hijackSession")
	}
	irc, err = hbot.NewBot(*serv, *nick, saslOption, hijackSession)
	if err != nil {
		log.LogPanic("Failed to start IRC component")
	}
	setupKeybaseLinks()
	log.LogWarn("Calling irc.Run()")
	irc.Run()
	log.LogError("irc.Run() returned")
}
func setupKeybaseLinks() {
	log.LogWarn("Setting up keybase channel links")
	api, err := k.ChatList()
	if err != nil {
		log.LogError(fmt.Sprintf("Err was not nil from ChatList() in setupKeybaseLinks(), ```%+v```", err))
	}
	for _, s := range api.Result.Conversations {
		if s.Channel.MembersType == keybase.TEAM && s.Channel.Name == channel.Name && s.Channel.TopicName != channel.TopicName {
			addIrcTrigger(s.Channel.TopicName)
		}
	}
}

func addIrcTrigger(name string) {
	if name == "general" {
		return
	}
	name = strings.Replace(name, "#", "", -1)
	log.LogWarn(fmt.Sprintf("Setting up trigger for #%s", name))
	var botLessTrigger = hbot.Trigger{
		func(b *hbot.Bot, m *hbot.Message) bool {
			log.LogInfo(fmt.Sprintf("i[%s]: %s", m.From, m.Content))
			return m.From != botNick
		},
		func(b *hbot.Bot, m *hbot.Message) bool {
			if m.To == fmt.Sprintf("#%s", name) {
				if m.Content == "" {
					return false
				}
				sendChat(fmt.Sprintf("[%s]: %s", m.From, m.Content), name)
				log.LogDebug("Calling sendChat")
			}
			return false
		},
	}
	irc.Channels = append(irc.Channels, fmt.Sprintf("#%s", name))
	irc.Join(fmt.Sprintf("#%s", name))
	log.LogWarn(fmt.Sprintf("Adding trigger for #%s to bot", name))
	irc.AddTrigger(botLessTrigger)
	log.LogDebug(fmt.Sprintf("irc.Channels = %+v", irc.Channels))
	sendChat(fmt.Sprintf("# Connected to #%s!", name), name)

}
func handleMessage(api keybase.ChatAPI) {
	if api.Msg.Channel.Name != channel.Name {
		return
	}
	if api.Msg.Sender.Username == k.Username {
		log.LogDebug("Ignoring message from me")
		return
	}

	if api.Msg.Content.Type != "text" {
		log.LogDebug("Non-text message ignored.")
		return
	}
	msgSender := api.Msg.Sender.Username
	msgBody := api.Msg.Content.Text.Body
	log.LogInfo(fmt.Sprintf("k[%s]: %s", msgSender, msgBody))

	parts := strings.Split(msgBody, " ")
	if parts[0] == "#" {
		log.LogWarn("Comment detected and msg ignored")
		return
	}
	if api.Msg.Channel.TopicName == channel.TopicName {
		log.LogDebug(fmt.Sprintf("Message found in control channel %s == %s", api.Msg.Channel.TopicName, channel.TopicName))
		if parts[0] == fmt.Sprintf("@%s", k.Username) {
			log.LogDebug("I was tagged in a message in Control Channel")
			if len(parts) == 3 && parts[1] == "join" {
				log.LogDebug("Join command detected")
				addIrcTrigger(parts[2])
			}
		}
		return
	}
	irc.Msg(fmt.Sprintf("#%s", api.Msg.Channel.TopicName), fmt.Sprintf("[%s]: %s", msgSender, msgBody))
}

func sendChat(message string, chann string) {
	newChannel := channel
	newChannel.TopicName = chann
	chat := k.NewChat(newChannel)
	_, err := chat.Send(strings.Replace(message, botNick, "@here", -1))
	if err != nil {
		log.LogError(fmt.Sprintf("There was an error %+v", err))
	}
}

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
	dev      = false
	k        = keybase.NewKeybase()
	channel  keybase.Channel
	irc      *hbot.Bot
	linkName = "halium"
	botNick  = "keyrcbot"
	serv     = flag.String("server", "chat.freenode.net:6667", "hostname and port")
	nick     = flag.String("nick", botNick, "nickname for the bot")
	logOpts  = loggy.LogOpts{
		OutFile:   "irc.log",
		KBTeam:    "nightmarehaus.bots",
		KBChann:   "general",
		ProgName:  "irclink",
		Level:     3,
		UseStdout: true,
	}
	log = loggy.NewLogger(logOpts)
)

func main() {
	log.LogWarn("Starting bot")
	if !k.LoggedIn {
		log.LogPanic("You are not logged in.")
	}
	channel.MembersType = keybase.TEAM
	channel.Name = "keyrc"
	channel.TopicName = "general"
	sendChat("Link starting", "general")
	go setupIRC()
	k.Run(func(api keybase.ChatAPI) {
		handleMessage(api)
	})
}

func setupIRC() {
	var err error
	saslOption := func(bot *hbot.Bot) {
		bot.SASL = true
		bot.Password = os.Getenv("IRC_PASS") // TODO: Set this
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
		if s.Channel.MembersType == keybase.TEAM && s.Channel.Name == channel.Name && s.Channel.TopicName != "general" {
			addIrcTrigger(s.Channel.TopicName)
		}
	}
}
func addIrcTrigger(name string) {
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
		return
	}

	if api.Msg.Content.Type != "text" {
		return
	}
	msgSender := api.Msg.Sender.Username
	msgBody := api.Msg.Content.Text.Body
	log.LogInfo(fmt.Sprintf("k[%s]: %s", msgSender, msgBody))

	parts := strings.Split(msgBody, " ")
	if parts[0] == "#" {
		return
	}
	if len(parts) == 3 {
		if parts[0] == fmt.Sprintf("@%s", k.Username) {
			if parts[1] == "join" {
				if api.Msg.Channel.TopicName == "general" && parts[2] != "general" {
					addIrcTrigger(parts[2])
					return
				}
			}
		}
	}
	if api.Msg.Channel.TopicName == "general" {
		return
	}
	irc.Msg(fmt.Sprintf("#%s", api.Msg.Channel.TopicName), fmt.Sprintf("[%s]: %s", msgSender, msgBody))
}

func sendChat(message string, chann string) {
	channel.TopicName = chann
	chat := k.NewChat(channel)
	_, err := chat.Send(message)
	if err != nil {
		log.LogError(fmt.Sprintf("There was an error %+v", err))
	}
}

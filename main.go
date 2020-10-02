package main

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	conf     = config{}
	commands = cmdMap{}
)

func main() {
	err := getConfig("config.toml")
	if err != nil {
		fmt.Println("Unable to read config file:", err)
		return
	}
	dg, err := discordgo.New("Bot " + strings.Trim(conf.Token, "\n\t"))
	if err != nil {
		fmt.Println("Unable to initialize Discord session:", err)
		return
	}

	dg.AddHandler(messageCreate)

	// Nice repetition, bro
	commands.add("help", "abre a lista de comandos.", helpHandler)
	commands.add("sobre", "mostra autores, e como sistema está rodando.", aboutHandler)
	commands.add("invite", "entre no servidor de suporte.", inviteHandler)
	commands.add("c", "convida pessoas no mesmo canal de voz para uma partida.", codeHandler)
	commands.add("servers", "lista todos os servidores em que fui adicionado.", srvHandler)
	commands.add("ping", "veja se estou vivo!", pingHandler)

	err = dg.Open()
	if err != nil {
		fmt.Println("Unable to open session:", err)
		return
	}

	dg.UpdateStatus(0, fmt.Sprintf("%shelp | amongbot.tk", conf.Prefix))

	fmt.Println("Bot is up!")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, conf.Prefix) {
		// Maybe this should be in a function (handleCommand?)
		cmdArgs := strings.Split(m.Content, " ")
		cmdStr := cmdArgs[0]
		cmdStr = strings.Replace(cmdStr, conf.Prefix, "", 1)
		cmd := commands[cmdStr]
		if cmd != nil {
			cmd.cmd(cmdArgs[1:], s, m)
		}
		return
	}

	if containsUsr(m.Mentions, s.State.User) {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(
			"<a:load:758855839497977857> *Opa, precisa de ajuda? meu prefixo é **'%s'**, caso precise de ajuda utilize **'%shelp'***", conf.Prefix, conf.Prefix))
		return
	}

	// Maybe we should see if user is in a call, but doing so for every message would be
	// rather expensive
	if t, _ := regexp.Match("^[A-Z]{6}($| )", []byte(m.Content)); t {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(
			"<a:load:758855839497977857> <@%s> *Você sabia que temos um sistema de convite?* `%sc <código da sala>`", m.Author.ID, conf.Prefix))
		return
	}
}

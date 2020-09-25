package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

const prefix = "%"

type command func([]string, *discordgo.Session, *discordgo.MessageCreate)

var commands = map[string]command{
	"ping":  pingHandler,
	"about": aboutHandler,
	"c":     codeHandler,
}

func main() {
	token, err := ioutil.ReadFile("token")
	if err != nil {
		fmt.Println("Unable to read token:", err)
		return
	}
	dg, err := discordgo.New("Bot " + strings.Trim(string(token), " \n\t"))
	if err != nil {
		fmt.Println("Unable to initialize Discord session:", err)
		return
	}

	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("Unable to open session:", err)
		return
	}

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

	if strings.HasPrefix(m.Content, prefix) {
		cmdArgs := strings.Split(m.Content, " ")
		cmdStr := cmdArgs[0]
		cmdStr = strings.Replace(cmdStr, prefix, "", 1)
		cmd := commands[cmdStr]
		if cmd != nil {
			cmd(cmdArgs[1:], s, m)
		}
	}
}

func pingHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, ")/")
}

func codeHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(args) != 1 {
		s.ChannelMessageSend(m.ChannelID, "Favor enviar um codigo")
		return
	}

	if !maybeValidCode(args[0]) {
		s.ChannelMessageSend(m.ChannelID, "Favor enviar um codigo valido")
		return
	}

	vs := getVoiceState(s, m.Author.ID, m.GuildID)
	if vs == nil {
		s.ChannelMessageSend(m.ChannelID, "Você não esta em um canal de voz")
		return
	}

	chann, err := s.Channel(vs.ChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprint("Erro ao requisitar canal: ", err))
		return
	}

	callUsers := getCallMembers(s, vs.ChannelID, vs.GuildID)

	//	s.ChannelMessageSend(m.ChannelID,
	//		fmt.Sprintf("> <a:verificador:758830726920536085> Convite - %s\n> **Código:** %s\n||%s||",
	//			chann.Name, args[0], mentions(callUsers)))

	s.ChannelMessageSendComplex(m.ChannelID,
		&discordgo.MessageSend{
			Content: fmt.Sprintf("||%s||", mentions(callUsers)),
			Embed: &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("<a:verificador:758830726920536085> Convite - %s", chann.Name),
				Description: fmt.Sprintf("**Código:** %s", args[0])}})
}

func aboutHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	ms := runtime.MemStats{}
	runtime.ReadMemStats(&ms)
	s.ChannelMessageSendEmbed(m.ChannelID,
		&discordgo.MessageEmbed{
			Title: "Sobre mim",
			Description: fmt.Sprintf(
				"**Autor:** <@145199845685067776>\n"+
					"**Runtime:**\n"+
					"**- Sistema operacional/arquitetura:** %s/%s\n"+
					"**- Memória (heap, alocado):** %d\n",
				runtime.GOOS, runtime.GOARCH, ms.Alloc)})
}

func maybeValidCode(c string) bool {
	r, err := regexp.Match("^[A-Z]{6}$", []byte(c))
	if err != nil {
		fmt.Println("maybeValidCode:", err)
		return false
	}
	return r
}

func getVoiceState(s *discordgo.Session, uid, gid string) *discordgo.VoiceState {
	g, err := s.State.Guild(gid)
	if err != nil {
		fmt.Println("getVoiceChan: cannot get guild state:", err)
		return nil
	}
	for _, vs := range g.VoiceStates {
		if vs.UserID == uid {
			return vs
		}
	}
	return nil
}

func getCallMembers(s *discordgo.Session, cid, gid string) (usrs []string) {
	g, err := s.State.Guild(gid)
	if err != nil {
		fmt.Println("getCallMembers: cannot get guild state:", err)
		return nil
	}
	for _, vs := range g.VoiceStates {
		if vs.ChannelID == cid {
			usrs = append(usrs, vs.UserID)
		}
	}
	return
}

func mentions(usrs []string) (m string) {
	for _, u := range usrs {
		m = fmt.Sprintf("%s, <@%s>", m, u)
	}
	m = m[2:]
	return
}

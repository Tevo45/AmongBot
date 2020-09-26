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
	"time"

	"github.com/bwmarrin/discordgo"
)

//const prefix = "%"
const prefix = "λ"

type cmdFunc func([]string, *discordgo.Session, *discordgo.MessageCreate)
type command struct {
	cmd  cmdFunc
	help string
}

type cmdMap map[string]*command

var commands = cmdMap{}

func (c *cmdMap) add(cmd, help string, fn cmdFunc) {
	// Nice syntax, bro
	(*c)[cmd] = &command{
		cmd: fn,
		help: help}
}

func (c *cmdMap) alias(cmd, dest string) {
	(*c)[cmd] = (*c)[dest]
}

var rateLimiters = map[string]<-chan time.Time{}

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

	// Nice repetition, bro
	commands.add("help", "Ajuda ae po", helpHandler)
	commands.alias("ajuda", "help")
	commands.alias("comandos", "help")

	commands.add("ping", "Manda um oi ae", pingHandler)
	commands.alias("eae", "ping")

	commands.add("c", "Avisa o povo", codeHandler)

	err = dg.Open()
	if err != nil {
		fmt.Println("Unable to open session:", err)
		return
	}

dg.UpdateStatus(0, fmt.Sprintf("%shelp | amongbot.tk", prefix))

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
			cmd.cmd(cmdArgs[1:], s, m)
		}
		return
	}

	if containsUsr(m.Mentions, s.State.User) {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(
			"<a:load:758855839497977857> *Opa, precisa de ajuda? meu prefixo é **'%s'**, caso precise de ajuda utilize **'%sajuda'***", prefix, prefix))
		return
	}

	
	if t, _ := regexp.Match("^[A-Z]{6}($| )", []byte(m.Content)); t {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(
			"<a:load:758855839497977857> <@%s> Você sabia que temos um sistema de convite? `%sc <código da sala>`", m.Author.ID, prefix))
		return
	}
}

func pingHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, ")/")
}

func codeHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(args) != 1 {
		s.ChannelMessageSend(m.ChannelID,
			"<a:load:758855839497977857> *Acho que você esqueceu de adicionar o codigo, não?*")
		return
	}

	if !maybeValidCode(args[0]) {
		s.ChannelMessageSend(m.ChannelID,
			"<a:load:758855839497977857> *Uhhh, acredito que seu código não esteja correto.*")
		return
	}

	vs := getVoiceState(s, m.Author.ID, m.GuildID)
	if vs == nil {
		s.ChannelMessageSend(m.ChannelID,
			"<a:load:758855839497977857> *Você não esta em nenhum canal de voz.*")
		return
	}

	if rateLimiters[m.Author.ID] != nil {
		s.ChannelMessageSend(m.ChannelID,
			"<a:load:758855839497977857> *Você esta a enviar pedidos rapido demais, favor tentar novamente mais tarde.*")
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

	go func() {
		uid := m.Author.ID
		rateLimiters[uid] = time.After(10 * time.Second)
		<-rateLimiters[uid]
		delete(rateLimiters, uid)
	}()
}

func aboutHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	ms := runtime.MemStats{}
	runtime.ReadMemStats(&ms)
	s.ChannelMessageSendEmbed(m.ChannelID,
		&discordgo.MessageEmbed{
			Title:       "<a:verificador:758830726920536085> Sobre mim",
			Description: "**・ Developer:** <@145199845685067776>\n**・ UX:** <@508719784381382706>\n",
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:   "<a:runtime:758883655471857674> **Runtime:**",
					Inline: false,
					Value: fmt.Sprintf(
						"**・ Sistema Operacional/Arquitetura:** %s %s/%s\n"+
							"**・ Memória (heap, alocado):** %d\n",
						osEmoji(runtime.GOOS), runtime.GOOS, runtime.GOARCH, ms.Alloc)}}})
}

func stubHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "Ainda não po")
}

func helpHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	cmds := ""
	for k, cmd := range commands {
		cmds += fmt.Sprintf("**%s:** %s\n", k, cmd.help)
	}
	s.ChannelMessageSend(m.ChannelID, cmds)
}

/*** Utilities ***/

func maybeValidCode(c string) bool {
	r, err := regexp.Match("^[A-Za-z]{6}$", []byte(c))
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

func osEmoji(s string) string {
	emojis := map[string]string{
		"windows": "<:windwos:758861126271631390>",
		"linux":   "<:tux:758874037706948648>",
		"solaris": "<:solaris:758875213961232404>",
		"openbsd": "<:puffy:758875557235654657>",
		"netbsd":  "<:netbsd:758875679961514014>",
		//		"plan9":     "<:spaceglenda:758857214596874241>",
		"plan9":     "<:glenda:758886314438295553>",
		"freebsd":   "<:freebased:758864143792078910>",
		"dragonfly": "<:dragonfly:758865198941077535>",
		"darwin":    "<:applel:758863829764931625>"}

	if emojis[s] == "" {
		return "❓"
	}
	return emojis[s]
}

func containsUsr(l []*discordgo.User, k *discordgo.User) bool {
	for _, i := range l {
		if i.ID == k.ID {
			return true
		}
	}
	return false
}
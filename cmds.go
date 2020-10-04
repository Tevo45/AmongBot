package main

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

/**/

type cmdFunc func([]string, *discordgo.Session, *discordgo.MessageCreate)
type command struct {
	cmd  cmdFunc
	help string
}

type cmdMap map[string]*command

func (c *cmdMap) add(cmd, help string, fn cmdFunc) {
	// Nice syntax, bro
	(*c)[cmd] = &command{
		cmd:  fn,
		help: help}
}

func (c *cmdMap) alias(cmd, dest string) {
	(*c)[cmd] = (*c)[dest]
}

/*** Stub ***/

func stubHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "Ainda não po")
}

/*** Ping ***/

func pingHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, ")/ estou vivo!")
}

/*** Invite ***/

func inviteHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSendEmbed(m.ChannelID,
		&discordgo.MessageEmbed{
			Title: "<a:redbit:759943137581203527> Convite - Support Server",
			Description: fmt.Sprintf("<a:runtime:758883655471857674> *Support Server' AmongBot* \n"+
				"[**・ Entrar no Servidor**](%s)", conf.InviteUrl),
			Color: 0xC02000,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: "https://pdhl.s-ul.eu/FX37PeEg"}})
}

/*** Game code ***/

var (
	rateLimiters  = map[string]<-chan time.Time{} // UID -> Timer
	commandTimers = map[string]<-chan time.Time{} // Match code -> Timer
)

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

	if commandTimers[args[0]] != nil {
		msg, _ := s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("<a:load:758855839497977857> %s, um convite já foi criado, espere o mesmo expirar para executar o comando novamente.", m.Author.Mention()))
		s.ChannelMessageDelete(m.ChannelID, m.ID)
		go selfDestruct(s, msg, time.After(5*time.Second))
		return
	}

	if rateLimiters[m.Author.ID] != nil {
		s.ChannelMessageSend(m.ChannelID,
			"<a:load:758855839497977857> *Você está a enviar pedidos rápido demais, tente novamente mais tarde.*")
		return
	}

	vs := getVoiceState(s, m.Author.ID, m.GuildID)
	if vs == nil {
		s.ChannelMessageSend(m.ChannelID,
			"<a:load:758855839497977857> *Você não esta em nenhum canal de voz.*")
		return
	}

	chann, err := s.Channel(vs.ChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprint("Erro ao requisitar canal: ", err))
		return
	}

	callUsers := getCallMembers(s, vs.ChannelID, vs.GuildID)

	msg, err := s.ChannelMessageSendComplex(m.ChannelID,
		&discordgo.MessageSend{
			Content: fmt.Sprintf("||%s||", mentions(callUsers)),
			Embed: &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("<a:redbit:759943137581203527> Convite - %s", chann.Name),
				Color:       0xC02000,
				Description: fmt.Sprintf("**・Código:** %s"+"\n<:plusamong:761656218153779232>*O convite expirará em 120 segundos.*", args[0]),
			},
		})

	if err != nil {
		fmt.Printf("Error on sending game code message to channel %s: %s\n", m.ChannelID, err)
	} else {
		go selfDestruct(s, msg, regTimer(2*time.Minute, &commandTimers, args[0]))
	}

	regTimer(10*time.Second, &rateLimiters, m.Author.ID)
}

/*** About ***/

func aboutHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	ms := runtime.MemStats{}
	runtime.ReadMemStats(&ms)
	/* This looks rather messy */
	s.ChannelMessageSendEmbed(m.ChannelID,
		&discordgo.MessageEmbed{
			Title: "<a:redbit:759943137581203527> Sobre mim",
			Description: "**・ Developer:** <@145199845685067776>\n" +
				"**・ User Experience:** <@508719784381382706>\n",
			Color: 0xC02000,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: "https://pdhl.s-ul.eu/FX37PeEg",
			},
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:   "<a:runtime:758883655471857674> **Runtime:**",
					Inline: false,
					Value: fmt.Sprintf(
						"**・ Sistema Operacional/Arquitetura:** %s %s/%s\n"+
							"**・ Memória (heap, alocado):** %d\n",
						osEmoji(runtime.GOOS), runtime.GOOS, runtime.GOARCH, ms.Alloc),
				},
			},
		})
}

/*** Help ***/

func helpHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	cmds := ""
	for k, cmd := range commands {
		cmds += fmt.Sprintf("**・ %s:** *%s*\n", strings.Title(k), cmd.help)
	}
	s.ChannelMessageSendEmbed(m.ChannelID,
		&discordgo.MessageEmbed{
			Title: fmt.Sprintf("<a:redbit:759943137581203527> Comandos - %s'prefix",
				conf.Prefix),
			Description: cmds,
			Color:       0xC02000,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: "https://pdhl.s-ul.eu/FX37PeEg", // TODO Host this on commander
			},
		})
}

/*** Servers ***/

func srvHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	names := ""
	em := map[string]*discordgo.Emoji{}
	wrks := make([]chan string, 0)
	// TODO GIFs
	for _, g := range s.State.Guilds {
		chann := make(chan string, 1)
		wrks = append(wrks, chann)
		go func(c chan string, g *discordgo.Guild) {
			e := registerGuildEmoji(s, g)
			ret := ""

			if e != nil {
				em[g.ID] = e
				ret += e.MessageFormat()
			} else {
				ret += "<:default:761420942072610817>"
			}
			ret += fmt.Sprintf("** ∙ %s**\n", g.Name)
			c <- ret
		}(chann, g)
	}
	for _, c := range wrks {
		names += <-c
	}
	s.ChannelMessageSend(m.ChannelID, names)
	for srv, emoji := range em {
		// FIXME Paralelize removal
		err := s.GuildEmojiDelete(conf.EmoteGuild, emoji.ID)
		if err != nil {
			fmt.Printf("failed to remove listing emoji for guild %s: %s\n", srv, err)
		}
	}
}

func newSrvHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	reactionSlider(s, m.ChannelID, guildProvider(s), nil)
}

/*** Play ***/

func matchHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	vs := getVoiceState(s, m.Author.ID, m.GuildID)
	if vs == nil {
		s.ChannelMessageSend(m.ChannelID,
			"<a:load:758855839497977857> *Você não esta em nenhum canal de voz.*")
		return
	}
	inv, err := s.ChannelInviteCreate(vs.ChannelID,
		discordgo.Invite{
			MaxAge:    3600, // An hour
			MaxUses:   10,   // Rather arbitrary
			Temporary: false,
		})
	if err != nil {
		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("Não foi possivel criar um convte: %s", err))
		return
	}

	callUsers := getCallMembers(s, vs.ChannelID, vs.GuildID)
	players := len(callUsers) // FIXME Is there a better way to get number of people on voice chan?

	s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
		Title:       "<a:redbit:759943137581203527> **Matchmaking**\n",
		Description: fmt.Sprintf("%s está procurando mais gente para jogar!", m.Author.Mention()),
		Color:       0xC02000,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://pdhl.s-ul.eu/FX37PeEg"},
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name: fmt.Sprintf(
					"<a:runtime:758883655471857674> **Informações:**"),
				Inline: true,
				Value: fmt.Sprintf(
					"**・Canal:** <#%s> "+"| **Players:** %d/10\n"+"<:aponta:761444193906065418> [Juntar-se ao grupo!](https://discord.gg/%s)",
					vs.ChannelID, players, inv.Code),
			},
		},
	})
}

func playChanHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	
}

/*** ***/

func testMenuHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	stubMenu := []menuEntry{}
	v := 24
	if len(args) != 0 {
		v, _ = strconv.Atoi(args[0])
	}
	for c := 0; c < v; c++ {
		stubMenu = append(stubMenu, stubItem{})
	}
	reactionSlider(s, m.ChannelID, sliceMenu(stubMenu), styleFunc(
		func(s *discordgo.MessageEmbed, p, np int) {
			s.Color = 0xFF0000
		}))
	//	if err != nil {
	//		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("error: %s", err))
	//	}
}

func premiumHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(args) < 0 {
		// Add stuff
	}
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("premium servers: %v", state.GetPremiumGuilds()))
}

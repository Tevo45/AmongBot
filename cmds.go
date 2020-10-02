package main

import (
	"fmt"
	"runtime"
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

var rateLimiters = map[string]<-chan time.Time{}

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
			"<a:load:758855839497977857> *Você está a enviar pedidos rápido demais, tente novamente mais tarde.*")
		return
	}

	chann, err := s.Channel(vs.ChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprint("Erro ao requisitar canal: ", err))
		return
	}

	callUsers := getCallMembers(s, vs.ChannelID, vs.GuildID)

	s.ChannelMessageSendComplex(m.ChannelID,
		&discordgo.MessageSend{
			Content: fmt.Sprintf("||%s||", mentions(callUsers)),
			Embed: &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("<a:redbit:759943137581203527> Convite - %s", chann.Name),
				Color:       0xC02000,
				Description: fmt.Sprintf("**・Código:** %s", args[0])}})

	go func() {
		uid := m.Author.ID
		rateLimiters[uid] = time.After(10 * time.Second)
		<-rateLimiters[uid]
		delete(rateLimiters, uid)
	}()
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
				URL: "https://pdhl.s-ul.eu/FX37PeEg",
			},
		})
}

/*** Servers ***/

func srvHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	names := ""
	em := map[string]*discordgo.Emoji{}
	// TODO Make this parallel
	// TODO GIFs
	for _, g := range s.State.Guilds {
		e := registerGuildEmoji(s, g)
		if e != nil {
			em[g.ID] = e
			names += e.MessageFormat()
		}
		names += fmt.Sprintf("** ∙ %s**\n", g.Name)
	}
	s.ChannelMessageSend(m.ChannelID, names)
	for srv, emoji := range em {
		err := s.GuildEmojiDelete(conf.EmoteGuild, emoji.ID)
		if err != nil {
			fmt.Printf("failed to remove listing emoji for guild %s: %s\n", srv, err)
		}
	}
}
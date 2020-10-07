package main

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

/**/

var commands = cmdMap{cmds: map[string]*command{}, aliases: map[string]*command{}}

type cmdFunc func([]string, *discordgo.Session, *discordgo.MessageCreate)
type command struct {
	cmd  cmdFunc
	help string
}

type cmdMap struct {
	cmds, aliases map[string]*command
}

func (c *cmdMap) Add(cmd, help string, fn cmdFunc) {
	c.cmds[cmd] = &command{
		cmd:  fn,
		help: help,
	}
}

func (c *cmdMap) Alias(cmd, dest string) {
	c.aliases[cmd] = c.cmds[dest]
}

func (c *cmdMap) Aliases(cmd string) (als []string) {
	als = []string{}
	com := c.Get(cmd)
	if com == nil {
		return
	}
	for k, i := range c.aliases {
		if i == com {
			als = append(als, k)
		}
	}
	return
}

func (c *cmdMap) Get(cmd string) *command {
	if com := c.cmds[cmd]; com != nil {
		return com
	}
	return c.aliases[cmd]
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
		cmd := commands.Get(cmdStr)
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

/*** Premium ***/

func prmHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSendEmbed(m.ChannelID,
		&discordgo.MessageEmbed{
			Title:       "<a:premium:762481446199361607> AmongBot' Premium Update",
			Description: fmt.Sprintf("• *Deseja saber as informações de como se tornar premium* \n*e quais as vantagens? você pode me chamar em:* <@508719784381382706>. \n• *Ou entrar em nosso servidor de support utilizando o comando %sinvite*", conf.Prefix),
			Color:       0xFFCC00,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: "https://pdhl.s-ul.eu/5902ngBb"}})
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
	for k, cmd := range commands.cmds {
		cmds += fmt.Sprintf("**・ %s:** *%s*", strings.Title(k), cmd.help)
		if aliases := commands.Aliases(k); len(aliases) > 0 {
			cmds += "\n\t**Aliases:** *"
			for _, alias := range aliases {
				cmds += conf.Prefix + alias + "*, *"
			}
			cmds = cmds[:len(cmds)-3]
		}
		cmds += "\n"
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
	reactionSlider(s, m.ChannelID, guildProvider(s), styleFunc(
		func(s *discordgo.MessageEmbed, p, np int) {
			s.Title = "<a:redbit:759943137581203527> **Servidores'**"
			s.Color = 0xC02000
		}))
}

/*** Play ***/

func matchHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	pc := state.GetGuildPrefs(m.GuildID).PlayChan
	if _, err := s.Channel(pc); err != nil {
		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("<a:load:758855839497977857> *Você precisa setar um canal primeiro, utilize* `%splaychan <canal>`", conf.Prefix))
		return
	}
	if m.ChannelID != pc {
		msg, _ := s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("%s <#%s>", m.Author.Mention(), pc))
		if msg != nil {
			go selfDestruct(s, msg, time.After(15*time.Second))
		}
		return
	}
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
	can, err := MemberHasPermission(s, m.GuildID, m.Author.ID,
		discordgo.PermissionManageChannels|discordgo.PermissionAdministrator|discordgo.PermissionManageServer)
	guild, err2 := s.Guild(m.GuildID)
	if err != nil || err2 != nil {
		s.ChannelMessageSend(m.ChannelID, "<a:load:758855839497977857> *Ops, parece que houve um erro, por favor tente novamente.*")
		return
	}
	if guild.OwnerID != m.Author.ID && !can {
		s.ChannelMessageSend(m.ChannelID, "<a:load:758855839497977857> *Uhhh, acho que você não tem permissão para fazer isso.*")
		return
	}
	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, "<a:load:758855839497977857> *Uhh, acredito que você tenha esquecido de adicionar um canal valido.*")
		return
	}
	var chann *discordgo.Channel
	if mcs := m.MentionChannels; len(mcs) != 0 {
		chann = mcs[0]
	}
	if chann == nil {
		var id uint64
		fmt.Sscanf(args[0], "<#%d>", &id)
		chann, _ = s.Channel(strconv.FormatUint(id, 10))
	}
	if chann == nil {
		chann, _ = s.Channel(args[0])
	}
	if chann == nil {
		s.ChannelMessageSend(m.ChannelID, "<a:load:758855839497977857> *Por favor, informe o canal corretamente.*")
		return
	}
	state.GetGuildPrefs(m.GuildID).PlayChan = chann.ID
	s.ChannelMessageSend(m.ChannelID, "<a:load:758855839497977857> *Canal adicionado com sucesso!*")
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
			s.Color = 0xC020000
		}))
	//	if err != nil {
	//		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("error: %s", err))
	//	}
}

func premiumHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(args) == 2 {
		state.Premium.Servers = append(state.Premium.Servers, premiumMembership{args[0], args[1]})
		return
	}
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("premium servers: %v", state.GetPremiumGuilds()))
}

func saveHandler(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	err := saveState()
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error while saving state: %s", err))
	}
}

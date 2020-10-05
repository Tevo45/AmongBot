package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	err := getConfig("config.toml")
	if err != nil {
		fmt.Println("Unable to read config file:", err)
		return
	}
	fmt.Printf("%v\n", loadState()) // TODO Error checking
	dg, err := discordgo.New("Bot " + strings.Trim(conf.Token, "\n\t"))
	if err != nil {
		fmt.Println("Unable to initialize Discord session:", err)
		return
	}

	dg.AddHandler(messageCreate)
	dg.AddHandler(rmReactionHandler)

	// Nice repetition, bro
	commands.Add("ajuda", "abre a lista de comandos.", helpHandler)
	commands.Alias("help", "ajuda")

	commands.Add("sobre", "mostra autores, e como sistema está rodando.", aboutHandler)
	commands.Alias("about", "sobre")

	commands.Add("suporte", "entre no servidor de suporte.", inviteHandler)
	commands.Alias("invite", "suporte")
	commands.Alias("support", "suporte")

	commands.Add("c", "convida pessoas no mesmo canal de voz para uma partida.", codeHandler)

	commands.Add("servers", "lista todos os servidores em que fui adicionado.", srvHandler)

	commands.Add("ping", "veja se estou vivo!", pingHandler)

	commands.Add("play", "cria um convite de matchmaking.", matchHandler)

	commands.Add("playchan", 
		"configura o canal aonde individuos podem utilizar"+conf.Prefix+"play", playChanHandler)
	commands.Alias("playset", "playchan")

	commands.Add("premium", "mostra as informações para se tornar premium.", prmHandler)


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

	saveState()
	dg.Close()
}

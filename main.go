package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/resterle/turnire-bot/discord"
	"github.com/resterle/turnire-bot/turniere"
)

// Registration page
// https://discord.com/oauth2/authorize?client_id=1203403155933503569&scope=bot&permissions=2048

// NOTE: Players list
// https://turniere.discgolf.de/index.php?p=events&sp=list-players&id=2124

const discordTokenName = "DISCORD_TOKEN"
const schduleInterval = time.Hour * 1
const notificationOffset = time.Hour * 10

func main() {
	token, tokenSet := os.LookupEnv(discordTokenName)
	if !tokenSet {
		log.Fatalf("%s needs to be set", discordTokenName)
	}

	bot, err := discord.InitBot(token)
	if err != nil {
		log.Fatalf("Error while creating bot. %s", err)
	}
	// Wait here until CTRL-C or other term signal is received.
	log.Println("Bot is now running.  Press CTRL-C to exit.")

	//go schedule(bot)()
	ticker := time.NewTicker(schduleInterval)
	exit := make(chan int, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	go startScheduling(ticker.C, exit, &wg, bot)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	log.Print("STOP")

	ticker.Stop()
	close(exit)

	// Cleanly close down the Discord session.
	bot.Close()
	wg.Wait()

	os.Exit(0)
}

func startScheduling(t <-chan time.Time, exit chan int, wg *sync.WaitGroup, bot *discord.Bot) {
	defer wg.Done()
	task(bot)
	for {
		select {
		case <-t:
			log.Print("Scheduling task")
			task(bot)
			continue
		case <-exit:
			log.Print("Stopping scheduler")
		}
		break
	}
}

func task(bot *discord.Bot) {
	reader, err := httpReader()
	if err == nil {
		tournaments := turniere.Parse(*reader)
		maybeSendMessages(bot, tournaments)
	}
}

func maybeSendMessages(bot *discord.Bot, tournaments []turniere.Turnament) {
	now := time.Now()
	for _, t := range tournaments {
		if t.RegistrationStartDate != nil {
			d := t.RegistrationStartDate.Sub(now)
			if d < notificationOffset && d > notificationOffset-schduleInterval {
				text := "⏰ Turnieranmeldung für:\n\"**%s**\"\nöffnet **heute um %s Uhr**\n🔗 %s"
				bot.SendMessage(fmt.Sprintf(text, t.Title, t.RegistrationStartDate.Format("15:04"), t.Link))
			}
		}
	}
}

// For development purposes
func fileReader() io.Reader {
	fi, err := os.Open("events.html")
	if err != nil {
		panic(err)
	}
	return fi
}

func httpReader() (*io.ReadCloser, error) {
	resp, err := http.Get("https://turniere.discgolf.de/index.php?p=events")
	if err != nil {
		log.Printf("Error while fetching tournament data from web. %s", err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error while fetching tournament data from web expected status code of %d got %d", http.StatusOK, resp.StatusCode)
		return nil, errors.New("status code error")
	}
	return &resp.Body, nil
}

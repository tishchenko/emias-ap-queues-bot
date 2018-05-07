package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"golang.org/x/net/proxy"
	"log"
	"net"
	"net/http"
	"context"
	"github.com/tishchenko/emias-ap-queues-bot/config"
	"time"
)

type ApBot struct {
	Bot   *tgbotapi.BotAPI
	Chats map[int64]int64
}

func NewApBot(conf *config.Config) *ApBot {
	bot := &ApBot{}
	bot.Chats = map[int64]int64{}

	var err error
	var client *http.Client

	if conf == nil {
		conf = &config.Config{TelApiToken: "585604919:AAG_wqdpDE5zg3bGJznhl0ZTVT2NqOpwyHs"}
	}

	if conf.Proxy != nil {
		auth := &proxy.Auth{conf.Proxy.User, conf.Proxy.Password}

		dialer, err := proxy.SOCKS5("tcp", conf.Proxy.IpAddr+":"+conf.Proxy.Port, auth, proxy.Direct)
		if err != nil {
			log.Panic(err)
		}
		transport := &http.Transport{}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
		client = &http.Client{Transport: transport}
	} else {
		client = &http.Client{}
	}

	bot.Bot, err = tgbotapi.NewBotAPIWithClient(conf.TelApiToken, client)
	if err != nil {
		log.Panic(err)
	}

	bot.Bot.Debug = true

	log.Printf("Authorized on account %s", bot.Bot.Self.UserName)

	return bot
}

func (bot *ApBot) Run() {

	c := make(chan int64)
	go bot.readQueuesStatFile(c)
	go bot.sendMessage(c)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.Bot.GetUpdatesChan(u)

	for update := range updates {

		if update.Message.Command() == "start" {
			bot.Chats[update.Message.Chat.ID] = update.Message.Chat.ID
		}
		if update.Message.Command() == "stop" {
			delete(bot.Chats, update.Message.Chat.ID);
		}

	}
}

func (bot *ApBot) sendMessage(c chan int64) {
	for {
		chatID := <-c
		msg := tgbotapi.NewMessage(chatID, "+")
		bot.Bot.Send(msg)
		print("+")
	}
}

func (bot *ApBot) readQueuesStatFile(c chan int64) {
	for {
		for chatID := range bot.Chats {
			c <- chatID
		}
		time.Sleep(time.Second)
	}
}

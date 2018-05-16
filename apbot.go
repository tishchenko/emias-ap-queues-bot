package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"golang.org/x/net/proxy"
	"log"
	"net"
	"net/http"
	"context"
	"github.com/tishchenko/emias-ap-queues-bot/config"
	"github.com/tishchenko/emias-ap-queues-bot/model"
	"time"
	"sort"
	"strconv"
)

type ApBot struct {
	Bot   *tgbotapi.BotAPI
	Chats map[int64]int64
	Model *model.Model
	AlarmLogic config.QueuesAlarmLogic
}

type QMesData struct {
	Name  string
	Type  string
	From  time.Time
	To    time.Time
	Delta int
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

	bot.Model = model.NewModelWithFileName(conf.ModelFileName)

	bot.Bot, err = tgbotapi.NewBotAPIWithClient(conf.TelApiToken, client)
	if err != nil {
		log.Panic(err)
	}

	bot.Bot.Debug = true

	log.Printf("Authorized on account %s", bot.Bot.Self.UserName)

	bot.AlarmLogic = *conf.AlarmLogic

	return bot
}

func (bot *ApBot) Run() {

	m := make(chan string)
	go bot.poll(m)
	go bot.sendMessage(m)

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

func (bot *ApBot) sendMessage(m chan string) {
	for {
		message := <-m
		for chatID := range bot.Chats {
			msg := tgbotapi.NewMessage(chatID, message)
			msg.ParseMode = tgbotapi.ModeHTML
			bot.Bot.Send(msg)
		}

		print("+")
	}
}

func (bot *ApBot) poll(m chan string) {
	for {
		m <- bot.readQueuesStatFile()
		time.Sleep(time.Duration(bot.AlarmLogic.PollInterval) * time.Second)
	}
}

func (bot *ApBot) readQueuesStatFile() string {
	bot.Model.Refresh()

	mesData := &[]QMesData{}

	bot.readQueueStat(mesData, bot.Model.NormalQueues, "")
	bot.readQueueStat(mesData, bot.Model.ExceptionQueues, "Exception Queue")

	return bot.generateStatMessage(*mesData)
}

func (bot *ApBot) readQueueStat(mesData *[]QMesData, queuesInfo map[string][]model.QueueInfo, queueType string) {
	for name, queues := range queuesInfo {
		sort.Slice(queues, func(i, j int) bool {
			return queues[i].DateTime.After(queues[j].DateTime)
		})

		if len(queues) < 2 || queues[0].Length == nil || queues[1].Length == nil {
			continue
		}

		delta := *queues[0].Length - *queues[1].Length

		var validDelta int
		if queueType == "" {
			validDelta = (*bot.AlarmLogic.NormalQueues)[name]
		} else {
			validDelta = (*bot.AlarmLogic.ExceptionQueues)[name]
		}

		if delta >= validDelta {
			*mesData = append(*mesData, QMesData{
				name,
				queueType,
				queues[1].DateTime,
				queues[0].DateTime,
				delta,
			})
		}

	}
}

func (bot *ApBot) generateStatMessage(mesData []QMesData) string {
	var mes string

	for _, m := range mesData {
		mes += "Очередь " +
			m.Type +
			" <b>" +
			m.Name +
			"</b> выросла на <b>" +
			strconv.Itoa(m.Delta) +
			"</b> в период с <b>" +
			m.From.Format("2006-01-02 15:04") +
			"</b> по <b>" +
			m.To.Format("2006-01-02 15:04") +
			"</b>\n"
	}

	return mes
}
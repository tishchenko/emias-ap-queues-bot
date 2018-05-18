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
	"encoding/json"
	"strings"
)

const version = "1.1.0"

type ApBot struct {
	Bot        *tgbotapi.BotAPI
	Chats      map[int64]int64
	Model      *model.Model
	AlarmLogic config.QueuesAlarmLogic
	InfoUrl    string
}

type QMesData struct {
	Name  string
	Type  string
	From  time.Time
	To    time.Time
	Delta int
}

var queueNames = []string{"APPOINTMENT", "SELF_APPOINTMENT", "UNMET_DEMAND", "AR_SCHEDULE_UPDATED"}
var queueTypes = []string{"", "EQ"}

func NewApBot(conf *config.Config) *ApBot {
	bot := &ApBot{}
	bot.Chats = map[int64]int64{}

	var err error
	var client *http.Client

	if conf == nil {
		conf = &config.Config{TelApiToken: "543460319:AAER5gKDmF0k3S6BNDhsm8v07iHspL8A4_M"}
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
	bot.InfoUrl = conf.InfoUrl

	return bot
}

func (bot *ApBot) Run() {

	m := make(chan string)
	go bot.poll(m)
	go bot.sendBroadcastMessage(m)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.Bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {

			switch update.Message.Command() {
			case "start":
				bot.Chats[update.Message.Chat.ID] = update.Message.Chat.ID
				bot.sendMessage(update.Message.Chat.ID, "Привет, "+update.Message.From.FirstName+" \xE2\x9C\x8C\n Мониторинг запущен.")
			case "stop":
				delete(bot.Chats, update.Message.Chat.ID)
				bot.sendMessage(update.Message.Chat.ID, "Ну, ты это, зови если что...")
				bot.sendMessage(update.Message.Chat.ID, "\xF0\x9F\x92\xA4")
			case "help":
				help := "Доступны команды:\n" +
					"/start - запуск мониторинга состояния очередей\n" +
					"/stop - приостановка мониторинга состояния очередей\n" +
					"/health - проверка бота на работоспособность\n" +
					"/queue (queueName [queueType]) - вывод статистики по очереди queueName (<b>" + strings.Join(queueNames, ", ") + "</b>) за последнее время; queueType - если не задан, то выводится статистика по normal queue, если <b>EQ</b>, то выводится статистика по exception queue\n"
					//"/rules - выводит правила оповещения, указанные в настройках"
				bot.sendMessage(update.Message.Chat.ID, help)
			case "health":
				bot.printHealth(update.Message.Chat.ID)
			case "queue":
				bot.printQueueStat(update.Message.Chat.ID, update.Message.CommandArguments())
			case "rules":
				bot.printRules(update.Message.Chat.ID)
			}
		}

	}
}

func (bot *ApBot) sendBroadcastMessage(m chan string) {
	for {
		message := <-m
		for chatID := range bot.Chats {
			bot.sendMessage(chatID, message)
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

	return bot.generateAlarmMessage(*mesData)
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

func (bot *ApBot) printQueueStat(chatID int64, queueName string) {

	queueType := ""
	posQueueNames := "Варианты: <b>" + strings.Join(queueNames, ", ") + "</b>"

	s := strings.Split(queueName, " ")
	if strings.TrimSpace(queueName) == "" {
		bot.sendMessage(chatID, "⛔ Не задано название очереди. "+posQueueNames)
		return
	}
	if !stringInSlice(s[0], queueNames) {
		bot.sendMessage(chatID, "⛔ Не знаю такой очереди! "+posQueueNames)
		return
	}
	if len(s) > 1 && !stringInSlice(s[1], queueTypes) {
		bot.sendMessage(chatID, "⛔ Не знаю такого типа очереди! Знаю только <b>EQ</b>")
		return
	} else {
		if len(s) > 1 {
			queueType = s[1]
		}
	}

	var queues map[string][]model.QueueInfo
	if queueType == "" {
		queues = bot.Model.NormalQueues
	} else {
		queues = bot.Model.ExceptionQueues
	}
	for name, queue := range queues {
		if name != s[0] {
			continue
		}
		bot.sendMessage(chatID, bot.generateStatMessage(queue))
	}
}

func (bot *ApBot) printRules(chatID int64) {
	rules, _ := json.Marshal(bot.AlarmLogic)
	bot.sendMessage(chatID, "<code>" + string(rules) + "</code>")
}

func (bot *ApBot) printHealth(chatID int64) {
	var mes string
	if intInSlice(chatID, bot.Chats) {
		mes = "❤ Мониторинг активен."
	} else {
		mes = "⛔ Мониторинг выключен! Для запуска выполните команду /start."
	}

	mes += "\nВсегда ваш <i>ApAggregateQueuesBot версии "+version+"</i>\n-=[ 🤖 ]=-"
	bot.sendMessage(chatID, mes)
}

func (bot *ApBot) generateAlarmMessage(mesData []QMesData) string {
	var mes string

	for _, m := range mesData {
		mes += "\xF0\x9F\x9A\xA8 Очередь " +
			m.Type +
			" <b>" +
			m.Name +
			"</b> выросла на <b>" +
			strconv.Itoa(m.Delta) +
			"</b> в период с <b>" +
			m.From.Format("2006-01-02 15:04") +
			"</b> по <b>" +
			m.To.Format("2006-01-02 15:04") +
			"</b>\n\n"
	}

	return mes
}

func (bot *ApBot) generateStatMessage(queueInfo []model.QueueInfo) string {

	mes := "📈 " +
		"Полная информация по ссылке " +
		"<a href=\"" +
		bot.InfoUrl +
		"\">" +
		bot.InfoUrl +
		"</a>\n<code>"

	for i, statInfo := range queueInfo {
		if i > 11 {
			break
		}
		mes += statInfo.DateTime.Format("2006-01-02 15:04") +
			"     " +
			strconv.Itoa(*statInfo.Length) +
			"\n"

	}
	mes += "</code>"

	return mes
}

func (bot *ApBot) sendMessage(chatID int64, message string) {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Bot.Send(msg)
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func intInSlice(a int64, list map[int64]int64) bool {
	for _, v := range list {
		if v == a {
			return true
		}
	}
	return false
}
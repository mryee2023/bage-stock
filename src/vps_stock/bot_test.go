package vps_stock

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
)

func TestTGBot(t *testing.T) {
	cli := resty.New().SetDebug(false)
	s, err := cli.R().SetBody(map[string]interface{}{
		"text": "test@" + time.Now().Format("2006-01-02 15:04:05"),
	}).
		Post("https://api.telegram.org/bot7770656473:AAE1D9k4rfPm4siew1qEpuIZfTcuuzgqVho/sendMessage?chat_id=-1002161428331")
	if err != nil {
		fmt.Printf("err: %v", err)
	}
	fmt.Print(s.String())
}

func TestTGBot2(t *testing.T) {
	bot, err := tgbotapi.NewBotAPI("7727116717:AAH31RbD5ygRkuWGO1EaCcfKybAoirykxaY")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	//bot.Send(tgbotapi.NewMessage(-1002398248297, "test"))

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID

			bot.Send(msg)
		}
	}
}

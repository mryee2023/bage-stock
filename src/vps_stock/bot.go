package vps_stock

import (
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

type BotNotifier interface {
	Notify(map[string]interface{})
}

type BarkNotifier struct {
	deviceToken string
}

func NewBarkNotifier(deviceToken string) *BarkNotifier {
	return &BarkNotifier{
		deviceToken: deviceToken,
	}
}

type TelegramNotifier struct {
	botToken string
	chatId   string
}

func NewTelegramNotifier(botToken string, chatId string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatId:   chatId,
	}
}

func (t *TelegramNotifier) Notify(msg map[string]interface{}) {

	log.WithField("msg", msg)
	msg["parse_mode"] = "Markdown"
	msg["disable_web_page_preview"] = false
	defer func() {
		CatchGoroutinePanic()
		log.Infof("telegram message sent")
	}()
	cli := resty.New().SetDebug(false)
	s, err := cli.R().SetBody(msg).
		Post("https://api.telegram.org/bot" + t.botToken + "/sendMessage?chat_id=" + t.chatId)
	if err != nil {
		log.WithField("err", err.Error())
		return
	}
	if s.StatusCode() != 200 {
		log.WithField("status", s.StatusCode())
	}
}

func (b *BarkNotifier) Notify(msg map[string]interface{}) {
	cli := resty.New().SetDebug(false)
	//msg["parse_mode"] = "MarkdownV2"
	s, err := cli.R().SetBody(msg).
		Post("https://api.day.app/" + b.deviceToken)
	if err != nil {
		log.Errorf("Error sending notification: %v", err)
		return
	}
	if s.StatusCode() != 200 {
		log.Errorf("Error sending notification: %d", s.StatusCode())
	}
}

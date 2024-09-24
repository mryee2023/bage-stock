package stock

import (
	"github.com/go-resty/resty/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
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
	tg := TgBotInstance()
	if tg == nil {
		return
	}
	defer func() {
		CatchGoroutinePanic()
	}()
	tgMsg := tgbotapi.NewMessage(cast.ToInt64(t.chatId), msg["text"].(string))
	tgMsg.ParseMode = tgbotapi.ModeMarkdown
	tg.Send(tgMsg)

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

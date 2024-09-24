package stock

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-module/carbon/v2"
	log "github.com/sirupsen/logrus"
	"github.com/zeromicro/go-zero/core/collection"
	"vps-stock/src/stock/vars"
)

var tgBot *tgbotapi.BotAPI

var startTime = time.Now()

func generateStartupMsg(title string, vps vars.VPS) string {

	startMsg := fmt.Sprintf("* %s: *\n\n", title)
	for _, product := range vps.Products {
		startMsg += fmt.Sprintf("> %s\n\n", product.Name)
		startMsg += fmt.Sprintf("   - %s\n\n", strings.Join(product.Kind, " , "))

	}
	startMsg += fmt.Sprintf("\n\n")
	return startMsg
}

func InitTgBotListen(token string) {

	defer func() {
		CatchGoroutinePanic()
	}()
	var err error
	tgBot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("tgbotapi init failure: %v", err)
	}

	tgBot.Debug = false
	tw, _ := collection.NewTimingWheel(time.Second, 120, func(key, value any) {
		if k, ok := key.(tgbotapi.DeleteMessageConfig); ok {
			rtn, err := tgBot.Request(k)
			if err != nil {
				log.WithField("msg", k).Errorf("delete message failure: %v", err)
			} else {
				log.WithField("msg", k).Infof("delete message rtn: %s", string(rtn.Result))
			}
		}
	})
	log.Infof("Authorized on account %s", tgBot.Self.UserName)

	go updates(tw)

}
func updates(tw *collection.TimingWheel) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := tgBot.GetUpdatesChan(u)
	var autoDeleteMsg = "\n\n⚠️消息5秒后自动删除"
	for update := range updates {
		if update.Message == nil {
			continue
		}
		if !update.Message.IsCommand() {
			continue
		}
		var msg tgbotapi.MessageConfig
		switch update.Message.Command() {
		//case "start":
		//	m := welcome() + autoDeleteMsg
		//
		//	msg = tgbotapi.NewMessage(update.Message.Chat.ID, m)
		case "status":
			cStart := carbon.CreateFromStdTime(startTime)
			m := fmt.Sprintf("启动时间: %s\n查询次数: %d", cStart.DiffForHumans(), atomic.LoadInt64(&TotalQuery))
			m += autoDeleteMsg
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, m)
		default:
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, "I don't know that command")
		}
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyToMessageID = update.Message.MessageID

		m, err := tgBot.Send(msg)
		if err != nil {
			log.Errorf("send message failure: %v", err)
		} else {
			tw.SetTimer(tgbotapi.DeleteMessageConfig{
				ChatID:    m.Chat.ID,
				MessageID: m.MessageID,
			}, "", time.Second*5)
			tw.SetTimer(tgbotapi.DeleteMessageConfig{
				ChatID:    update.Message.Chat.ID,
				MessageID: update.Message.MessageID,
			}, "", time.Second*5)

		}
	}
}

func TgBotInstance() *tgbotapi.BotAPI {
	return tgBot
}

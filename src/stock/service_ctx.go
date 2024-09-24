package stock

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"vps-stock/src/stock/db"
	"vps-stock/src/stock/vars"
)

type ServiceCtx struct {
	TgBotAPi *tgbotapi.BotAPI
	Config   *vars.Config
}

func NewServiceCtx(tgBotAPi *tgbotapi.BotAPI, config *vars.Config) *ServiceCtx {
	db.Open(config)
	return &ServiceCtx{
		TgBotAPi: tgBotAPi,
		Config:   config,
	}
}

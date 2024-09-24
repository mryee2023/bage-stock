package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"bage/src/vps_stock"
	"bage/src/vps_stock/vars"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var configFile = flag.String("f", "etc/config.yaml", "the config file")
var config vars.Config

func createBot() vps_stock.BotNotifier {
	platform := strings.TrimSpace(strings.ToLower(config.Notify.Platform))
	switch platform {
	case "bark":
		return vps_stock.NewBarkNotifier(config.Notify.Key)
	case "telegram":
		return vps_stock.NewTelegramNotifier(config.Notify.Key, config.Notify.ChatId)
	}
	return nil
}

func initBageVM(vps vars.VPS, notifier vps_stock.BotNotifier) {
	p := vps_stock.NewBageVpsStockNotifier(vps, notifier)
	d, e := time.ParseDuration(config.CheckTime)
	if e != nil {
		log.Fatalf("error: %v", e)
	}

	go func() {
		p.Notify()
		ticker := time.NewTicker(d)
		for {
			select {
			case <-ticker.C:
				p.Notify()
			default:
			}
		}
	}()
}

func initHaloVM(vps vars.VPS, notifier vps_stock.BotNotifier) {
	p := vps_stock.NewHaloVpsStockNotifier(vps, notifier)
	d, e := time.ParseDuration(config.CheckTime)
	if e != nil {
		log.Fatalf("error: %v", e)
	}

	go func() {
		p.Notify()
		ticker := time.NewTicker(d)
		for {
			select {
			case <-ticker.C:
				p.Notify()
			default:
			}
		}
	}()
}
func startupInfo() string {
	var startMsg string
	for _, vps := range config.VPS {
		if vps.Name == "bagevm" {
			startMsg += fmt.Sprintf("* BageVM: *\n\n")
			for _, product := range vps.Products {
				//startMsg += fmt.Sprintf("> %s\n\n", product.Name)
				startMsg += fmt.Sprintf("> %s\n\n", strings.Join(product.Kind, " , "))

			}
			startMsg += fmt.Sprintf("\n\n")
			continue
		}
		if vps.Name == "halo" {
			startMsg += fmt.Sprintf("* Halo: *\n\n")
			for _, product := range vps.Products {
				//startMsg += fmt.Sprintf("> %s\n\n", product.Name)
				startMsg += fmt.Sprintf("%s", strings.Join(product.Kind, "   \n\n"))
			}
			startMsg += fmt.Sprintf("\n\n")
			continue
		}
	}
	return startMsg
}
func initUpdates() {
	defer func() {
		vps_stock.CatchGoroutinePanic()
	}()
	bot, err := tgbotapi.NewBotAPI(config.Notify.Key)
	if err != nil {
		log.Fatalf("create tgbot failure: %v", err)
	}
	tgbotapi.NewSetMyCommands(tgbotapi.BotCommand{
		Command:     "info",
		Description: "显示启动信息",
	}, tgbotapi.BotCommand{
		Command:     "status",
		Description: "显示运行状态",
	})

	log.Infof("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message == nil { // ignore any non-Message updates
			continue
		}
		if !update.Message.IsCommand() { // ignore any non-command Messages
			continue
		}
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		switch update.Message.Command() {
		case "info":
			msg.Text = startupInfo()
		case "status":
			msg.Text = "running since " + startTime.Format("2006-01-02 15:04:05")
		default:
			msg.Text = "I don't know that command"
		}
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyToMessageID = update.Message.MessageID
		if _, err := bot.Send(msg); err != nil {
			log.WithField("command", update.Message.Command()).Errorf("send message failure: %v", err)
		}

		// delete command message after reply
		tout := time.NewTicker(5 * time.Second)
		update := update
		go func() {
			deleteMessageConfig := tgbotapi.DeleteMessageConfig{
				ChatID:    update.Message.Chat.ID,
				MessageID: update.Message.MessageID,
			}
			<-tout.C
			bot.Request(deleteMessageConfig)
		}()

	}
}

var startTime = time.Now()

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)

	flag.Parse()

	b, err := os.ReadFile(*configFile)
	if err != nil {
		fmt.Printf("加载配置文件异常")
		panic(err)
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	d, e := time.ParseDuration(config.Frozen)
	if e != nil {
		log.Fatalf("error: %v", e)
	}
	vps_stock.StartFrozen(d)

	bot := createBot()
	if bot == nil {
		log.Fatalf("error: invalid bot platform")
	}

	startMsg := "📢 VPS库存通知 已启动\n\n"

	for _, vps := range config.VPS {
		if vps.Name == "bagevm" {
			startMsg += fmt.Sprintf("* BageVM: *\n\n")
			for _, product := range vps.Products {
				startMsg += fmt.Sprintf("> %s\n\n", product.Name)
				startMsg += fmt.Sprintf("   - %s\n\n", strings.Join(product.Kind, " , "))

			}
			startMsg += fmt.Sprintf("\n\n")
			initBageVM(vps, bot)
			continue
		}
		if vps.Name == "halo" {
			startMsg += fmt.Sprintf("* Halo: *\n\n")
			for _, product := range vps.Products {
				startMsg += fmt.Sprintf("> %s\n\n", product.Name)
				startMsg += fmt.Sprintf("   - %s\n\n", strings.Join(product.Kind, " , "))
			}
			startMsg += fmt.Sprintf("\n\n")
			initHaloVM(vps, bot)
			continue
		}
	}

	startMsg += "当前设定的检查时间间隔为: *" + config.CheckTime + "* \n\n"
	startMsg += "当前设定的冻结时间为: *" + config.Frozen + "* \n\n"

	//bot.Notify(map[string]interface{}{
	//	"text": startMsg,
	//})
	fmt.Println("Listening......")
	go initUpdates()
	select {}
}

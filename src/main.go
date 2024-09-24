package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	_ "net/http/pprof"

	"github.com/fsnotify/fsnotify"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-module/carbon/v2"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"vps-stock/src/stock"
	"vps-stock/src/stock/vars"
)

var configFile = flag.String("f", "etc/config.yaml", "the config file")
var config vars.Config
var bot stock.BotNotifier
var (
	BageKindStock = make(map[string]int)
	HaloKindStock = make(map[string]int)
)

func createBot() {
	platform := strings.TrimSpace(strings.ToLower(config.Notify.Platform))
	switch platform {
	case "bark":
		bot = stock.NewBarkNotifier(config.Notify.Key)
	case "telegram":
		bot = stock.NewTelegramNotifier(config.Notify.Key, config.Notify.ChatId)
	}

}

// 监控配置文件变化
func watchConfig(filePath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	// 添加文件监控
	err = watcher.Add(filePath)
	if err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Infof("watcher event failure")
				return
			}
			if event.Has(fsnotify.Write) {
				b, err := os.ReadFile(filePath)
				if err != nil {
					log.Errorf("read config failure :%s, %v", filePath, err)
				}
				err = yaml.Unmarshal(b, &config)
				if err != nil {
					log.Fatalf("unmarshal config failure: %v", err)
				}
				log.WithField("event", event).WithField("file", filePath).Info("配置文件已重新加载")
				breakChan <- struct{}{}
				initVpsWatch()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.WithField("err", err.Error()).Info("配置文件被修改")
		}
	}
}

func startBageVM(vps vars.VPS, notifier stock.BotNotifier) {

	defer func() {
		stock.CatchGoroutinePanic()
	}()
	p := stock.NewBageVpsStockNotifier(vps, notifier, BageKindStock)
	p.Notify()

}

func startHaloVM(vps vars.VPS, notifier stock.BotNotifier) {
	defer func() {
		stock.CatchGoroutinePanic()
	}()

	p := stock.NewHaloVpsStockNotifier(vps, notifier, HaloKindStock)
	p.Notify()
}

func StartVpsWatch() {
	for _, vps := range config.VPS {
		if vps.Name == "bagevm" {
			startBageVM(vps, bot)
			continue
		}
		if vps.Name == "halo" {
			startHaloVM(vps, bot)
			continue
		}
	}
}

var breakChan = make(chan struct{})

func initVpsWatch() {
	d, e := time.ParseDuration(config.CheckTime)
	if e != nil {
		log.Fatalf("error: %v", e)
	}
	log.Infof("init vps watching...... %s", config.CheckTime)
	go func() {
		ticker := time.NewTicker(d)
		defer func() {
			ticker.Stop()
			stock.CatchGoroutinePanic()
		}()
		for {
			select {
			case <-ticker.C:
				StartVpsWatch()
			case <-breakChan:
				return
			}
		}
	}()
}

func getAbsolutePath() string {
	// 获取当前可执行文件的路径
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("os.Executable() failed: %v", err)
	}
	// 获取绝对路径
	return filepath.Dir(exe)
}

func generateStartupMsg(title string, vps vars.VPS) string {

	startMsg := fmt.Sprintf("* %s: *\n\n", title)
	for _, product := range vps.Products {
		startMsg += fmt.Sprintf("> %s\n\n", product.Name)
		startMsg += fmt.Sprintf("   - %s\n\n", strings.Join(product.Kind, " , "))

	}
	startMsg += fmt.Sprintf("\n\n")
	return startMsg
}

func initTgBotUpdates() {
	defer func() {
		stock.CatchGoroutinePanic()
	}()
	bot, err := tgbotapi.NewBotAPI(config.Notify.Key)
	if err != nil {
		log.Fatalf("tgbotapi init failure: %v", err)
	}

	bot.Debug = false

	log.Infof("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		if !update.Message.IsCommand() {
			continue
		}
		var msg tgbotapi.MessageConfig

		switch update.Message.Command() {
		case "start":
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, welcome())
		case "status":
			cStart := carbon.CreateFromStdTime(startTime)
			m := fmt.Sprintf("启动时间: %s\n查询次数: %d", cStart.DiffForHumans(), atomic.LoadInt64(&stock.TotalQuery))
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, m)
		default:
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, "I don't know that command")
		}
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyToMessageID = update.Message.MessageID

		m, err := bot.Send(msg)
		if err != nil {
			log.Errorf("send message failure: %v", err)
		} else {
			go func() {
				defer func() {
					stock.CatchGoroutinePanic()
				}()
				ticker := time.NewTicker(time.Second * 5)
				defer ticker.Stop()
				select {
				case <-ticker.C:
					bot.Request(tgbotapi.DeleteMessageConfig{
						ChatID:    m.Chat.ID,
						MessageID: m.MessageID,
					})
					bot.Request(tgbotapi.DeleteMessageConfig{
						ChatID:    update.Message.Chat.ID,
						MessageID: update.Message.MessageID,
					})
				}
			}()
		}
	}
	fmt.Println("initTgBotUpdates end")
}

var startTime = time.Now()

func welcome() string {
	var startMsg string
	for _, vps := range config.VPS {
		if vps.Name == "bagevm" {
			startMsg += generateStartupMsg("BageVM", vps)
			continue
		}
		if vps.Name == "halo" {
			startMsg += generateStartupMsg("HaloCloud", vps)
			continue
		}
	}
	return startMsg
}
func main() {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		PrettyPrint:     true,
	})
	log.SetLevel(log.InfoLevel)
	carbon.SetDefault(carbon.Default{
		Layout:       carbon.DateTimeLayout,
		Timezone:     carbon.PRC,
		WeekStartsAt: carbon.Monday,
		Locale:       "zh-CN",
	})
	file, err := os.OpenFile(filepath.Join(getAbsolutePath(), "stock.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	log.SetOutput(file)
	//log.SetOutput(os.Stdout)
	flag.Parse()

	b, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("load config failure :%s, %v", *configFile, err)
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		log.Fatalf("unmarshal config failure: %v", err)
	}

	createBot()
	if bot == nil {
		log.Fatalf("error: invalid bot platform")
	}

	go func() {
		defer func() {
			stock.CatchGoroutinePanic()
		}()
		watchConfig(*configFile)
	}()

	initVpsWatch()

	go func() {
		defer func() {
			stock.CatchGoroutinePanic()
		}()
		initTgBotUpdates()
	}()
	// 定义路由
	http.HandleFunc("/log", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		http.ServeFile(writer, request, filepath.Join(getAbsolutePath(), "stock.log"))
	})
	log.Infof("Service start success,Listen On :9527......")
	if err := http.ListenAndServe(":9527", nil); err != nil {
		log.Fatalf("start web server failure : %v", err)
	}

}

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"vps-stock/src/stock"
	"vps-stock/src/stock/vars"
)

var configFile = flag.String("f", "etc/config.yaml", "the config file")
var config vars.Config

func createBot() stock.BotNotifier {
	platform := strings.TrimSpace(strings.ToLower(config.Notify.Platform))
	switch platform {
	case "bark":
		return stock.NewBarkNotifier(config.Notify.Key)
	case "telegram":
		return stock.NewTelegramNotifier(config.Notify.Key, config.Notify.ChatId)
	}
	return nil
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
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.WithField("event", event).WithField("file", filePath).Info("配置文件被修改")

			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.WithField("err", err.Error()).Info("配置文件被修改")
		}
	}
}

func initBageVM(vps vars.VPS, notifier stock.BotNotifier) {
	p := stock.NewBageVpsStockNotifier(vps, notifier)
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

func initHaloVM(vps vars.VPS, notifier stock.BotNotifier) {
	p := stock.NewHaloVpsStockNotifier(vps, notifier)
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

func main() {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		PrettyPrint:     true,
	})
	log.SetLevel(log.InfoLevel)

	file, err := os.OpenFile(filepath.Join(getAbsolutePath(), "stock.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	log.SetOutput(file)

	flag.Parse()

	b, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("load config failure :%s, %v", *configFile, err)
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		log.Fatalf("unmarshal config failure: %v", err)
	}

	d, e := time.ParseDuration(config.Frozen)
	if e != nil {
		log.Fatalf("parse frozen duration failure: %s, %v", config.Frozen, e)
	}
	stock.StartFrozen(d)

	bot := createBot()
	if bot == nil {
		log.Fatalf("error: invalid bot platform")
	}

	startMsg := "📢 VPS库存通知 已启动\n\n"
	go func() {
		defer func() {
			stock.CatchGoroutinePanic()
		}()
		watchConfig(*configFile)
	}()

	for _, vps := range config.VPS {
		if vps.Name == "bagevm" {
			startMsg += generateStartupMsg("BageVM", vps)
			initBageVM(vps, bot)
			continue
		}
		if vps.Name == "halo" {
			startMsg += generateStartupMsg("HaloCloud", vps)
			initHaloVM(vps, bot)
			continue
		}
	}

	startMsg += "当前设定的检查时间间隔为: *" + config.CheckTime + "* \n\n"
	startMsg += "当前设定的冻结时间为: *" + config.Frozen + "* \n\n"

	bot.Notify(map[string]interface{}{
		"text": startMsg,
	})

	// 定义路由
	http.HandleFunc("/log", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		http.ServeFile(writer, request, filepath.Join(getAbsolutePath(), "stock.log"))
	})

	if err := http.ListenAndServe(":9527", nil); err != nil {
		log.Fatalf("start web server failure : %v", err)
	}
	fmt.Println("Listen On :9527......")
	select {}
}

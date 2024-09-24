package main

import (
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "net/http/pprof"

	"github.com/fsnotify/fsnotify"
	"github.com/golang-module/carbon/v2"
	log "github.com/sirupsen/logrus"
	"github.com/zeromicro/go-zero/core/proc"
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

func main() {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		PrettyPrint:     false,
	})
	log.SetLevel(log.InfoLevel)
	carbon.SetDefault(carbon.Default{
		Layout:       carbon.DateTimeLayout,
		Timezone:     carbon.PRC,
		WeekStartsAt: carbon.Monday,
		Locale:       "zh-CN",
	})

	defer func() {
		stock.CatchGoroutinePanic()
	}()

	file, err := os.OpenFile(filepath.Join(getAbsolutePath(), "stock.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	//log.SetOutput(file)
	//log.SetOutput(os.Stdout)
	log.SetOutput(io.MultiWriter(file, os.Stdout))
	flag.Parse()

	b, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("load config failure :%s, %v", *configFile, err)
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		log.Fatalf("unmarshal config failure: %v", err)
	}

	bot = stock.NewTelegramNotifier(config.Notify.Key, config.Notify.ChatId)
	if bot == nil {
		log.Fatalf("error: invalid bot platform")
	}

	proc.AddShutdownListener(func() {
		bot.Notify(stock.NotifyMessage{Text: "⚠️ BageVM 库存监控服务已停止", ChatId: &stock.AlertId})
		log.Info("service shutdown")
	})

	go func() {
		defer func() {
			stock.CatchGoroutinePanic()
		}()
		watchConfig(*configFile)
	}()

	initVpsWatch()

	stock.InitTgBotListen(config.Notify.Key)

	bot.Notify(stock.NotifyMessage{Text: "📢 BageVM 库存监控服务已启动", ChatId: &stock.AlertId})

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

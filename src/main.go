package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

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
	stock.StartFrozen(d)

	bot := createBot()
	if bot == nil {
		log.Fatalf("error: invalid bot platform")
	}

	startMsg := "📢 VPS库存通知 已启动\n\n"

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
	fmt.Println("Listening......")
	select {}
}

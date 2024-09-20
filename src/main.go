package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"bage/src/bage"
	"bage/src/bage/vars"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
)

var cli *resty.Client
var notified map[string]time.Time
var mapLock sync.Mutex

func start() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
	cli = resty.New()
	notified = make(map[string]time.Time)
	frozen, e := time.ParseDuration(config.Frozen)
	if e != nil {
		frozen = 30 * time.Minute
	}
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for {
			select {
			case <-ticker.C:
				mapLock.Lock()
				for k, v := range notified {
					if time.Since(v) > frozen {
						delete(notified, k)
						log.WithField("product", k).Info("release frozen")
					}
				}
				mapLock.Unlock()
			}
		}
	}()
}

type VpsStockNotifier interface {
	Notify()
}

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

	defer func() {
		bage.CatchGoroutinePanic()
		log.Infof("telegram message sent")
	}()
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
	cli := resty.New()
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

type BageVpsStockNotifier struct {
	baseUrl  string
	products []vars.Product
	bot      BotNotifier
}

func NewBageVpsStockNotifier(baseUrl string, products []vars.Product, bot BotNotifier) *BageVpsStockNotifier {
	return &BageVpsStockNotifier{
		baseUrl:  baseUrl,  // "https://www.bagevm.com/index.php?rp=/store/",
		products: products, // []string{"tw-hinet-vds"}, // "los-angeles-servers", "hong-kong-servers", "singapore-servers", "japan-servers", "united-kingdom-servers", "germany-servers"},
		bot:      bot,
	}
}

func openBrowser(url string) error {
	defer func() {
		bage.CatchGoroutinePanic()
	}()
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", url)
	case "linux": // Linux
		cmd = exec.Command("xdg-open", url)
	case "windows": // Windows
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("不支持的操作系统")
	}
	return cmd.Start()
}

func (b *BageVpsStockNotifier) Notify() {
	if len(b.products) == 0 {
		return
	}
	if len(b.baseUrl) == 0 {
		return
	}
	cli.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")
	cli.Header.Add("Referer", "https://www.bagevm.com/index.php")
	var wg sync.WaitGroup
	var items []*vars.VpsStockItem
	var mu sync.Mutex
	for _, product := range b.products {
		u := b.baseUrl + product.Name
		wg.Add(1)

		go func() {
			defer wg.Done()
			res, err := cli.R().Get(u)

			if err != nil {
				log.WithField("url", u).Errorf("Error fetching url: %v", err)
				return
			}
			if res.StatusCode() != 200 {
				log.WithField("status", res.StatusCode()).Error("Error fetching url")
				return
			}
			v := b.parseResponse(product.Kind, res.String())
			if v != nil {
				mu.Lock()
				items = append(items, v...)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	var body string
	var sendMsg bool
	for _, item := range items {
		if item.Available > 0 {
			if _, ok := notified[item.ProductName]; ok {
				continue
			}
			sendMsg = true
			body += fmt.Sprintf("%s: 库存 %d\n\n", item.ProductName, item.Available)
			body += fmt.Sprintf("购买链接: %s\n\n", item.BuyUrl)
			mapLock.Lock()
			notified[item.ProductName] = time.Now()
			mapLock.Unlock()
		}
	}
	if sendMsg {
		b.bot.Notify(map[string]interface{}{
			"title": "BageVM库存通知",
			"body":  body,
			"group": "BageVM",
			"text":  body,
		})
	}
}
func (b *BageVpsStockNotifier) parseResponse(kind []string, body string) []*vars.VpsStockItem {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))

	if err != nil {
		log.Errorf("Error parsing response: %v", err)
		return nil
	}

	var rtn []*vars.VpsStockItem

	doc.Find("#productspo  div.col-md-3").Each(func(i int, s *goquery.Selection) {
		h5 := s.Find("div.proprice>h5")

		productName := h5.Contents().Not("em").Text()
		productName = strings.TrimSpace(productName)
		if len(kind) > 0 {
			var found bool
			for _, k := range kind {
				if strings.Contains(strings.ToLower(productName), strings.ToLower(k)) {
					found = true
					break
				}
			}
			if !found {
				return
			}
		}
		item := &vars.VpsStockItem{
			ProductName: productName,
			Available:   9999,
		}
		if h5.Find("em").Length() > 0 {
			// 2. 获取 <em> 标签内 <span> 的值
			available := h5.Find("em span").Text()
			available = strings.Replace(available, "Available", "", -1)
			available = strings.TrimSpace(available) // 去掉多余的空格
			item.Available = cast.ToInt(available)
			if item.Available == 0 {
				return
			}
		} else {
			item.Available = 9999
		}
		// 3. 获取购买链接

		buyUrl, _ := s.Find("div.proprice a.btn").Attr("href")
		item.BuyUrl = "https://www.bagevm.com/" + buyUrl
		rtn = append(rtn, item)
	})
	return rtn
}

var configFile = flag.String("f", "etc/config.yaml", "the config file")
var config vars.Config

func createBot() BotNotifier {
	platform := strings.TrimSpace(strings.ToLower(config.Notify.Platform))
	switch platform {
	case "bark":
		return NewBarkNotifier(config.Notify.Key)
	case "telegram":
		return NewTelegramNotifier(config.Notify.Key, config.Notify.ChatId)
	}
	return nil
}

func main() {

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

	start()

	bot := createBot()
	if bot == nil {
		log.Fatalf("error: invalid bot platform")
	}
	p := NewBageVpsStockNotifier(config.VPS.BaseURL+"/index.php?rp=/store/", config.VPS.Products, bot)
	d, e := time.ParseDuration(config.CheckTime)
	if e != nil {
		log.Fatalf("error: %v", e)
	}
	p.Notify()
	go func() {
		ticker := time.NewTicker(d)
		for {
			select {
			case <-ticker.C:
				p.Notify()
			default:
			}
		}
	}()

	fmt.Println("starting...")
	select {}
}

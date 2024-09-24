package vps_stock

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"bage/src/vps_stock/vars"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

//var cli *resty.Client
//
//func init() {
//	cli = resty.New().SetDebug(false)
//}

type VpsStockNotifier interface {
	Notify()
}

type BageVpsStockNotifier struct {
	cli *resty.Client
	vps vars.VPS
	bot BotNotifier
}

func NewBageVpsStockNotifier(vps vars.VPS, bot BotNotifier) *BageVpsStockNotifier {
	return &BageVpsStockNotifier{
		vps: vps,
		bot: bot,
		cli: resty.New(),
	}
}

func (b *BageVpsStockNotifier) Notify() {
	if len(b.vps.Products) == 0 {
		return
	}
	if len(b.vps.BaseURL) == 0 {
		return
	}
	b.cli.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")
	b.cli.Header.Add("Referer", "https://www.bagevm.com/index.php")
	var wg sync.WaitGroup
	var items []*vars.VpsStockItem
	var mu sync.Mutex
	for _, product := range b.vps.Products {
		u := b.vps.ProductUrl + product.Name
		wg.Add(1)

		go func() {
			defer wg.Done()
			res, err := b.cli.R().Get(u)

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
	var body = "📢 *BageVM 库存通知*\n\n"
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
		item.BuyUrl = b.vps.BaseURL + "/" + buyUrl
		item.BuyUrl = fmt.Sprintf("[%s](%s)", item.BuyUrl, item.BuyUrl)

		rtn = append(rtn, item)
	})
	return rtn
}

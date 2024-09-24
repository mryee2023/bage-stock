package stock

import (
	"fmt"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"vps-stock/src/stock/vars"
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
	cli       *resty.Client
	vps       vars.VPS
	bot       BotNotifier
	kindStock map[string]int
}

func NewBageVpsStockNotifier(vps vars.VPS, bot BotNotifier, kindStock map[string]int) *BageVpsStockNotifier {
	cli := resty.New().SetDebug(false)
	cli.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")
	cli.Header.Add("Referer", vps.BaseURL)
	return &BageVpsStockNotifier{
		vps:       vps,
		bot:       bot,
		cli:       cli,
		kindStock: kindStock,
	}
}

func (b *BageVpsStockNotifier) Notify() {
	if len(b.vps.Products) == 0 {
		return
	}
	if len(b.vps.BaseURL) == 0 {
		return
	}
	defer func() {
		CatchGoroutinePanic()
	}()
	var wg sync.WaitGroup
	var items []*vars.VpsStockItem
	var mu sync.Mutex
	for _, product := range b.vps.Products {
		u := b.vps.ProductUrl + product.Name
		log.WithField("url", u).Trace("[bage] fetching url")
		wg.Add(1)

		go func() {
			defer func() {
				wg.Done()
				CatchGoroutinePanic()

			}()
			res, err := b.cli.R().Get(u)

			if err != nil {
				log.WithField("url", u).Warn("[bage] Error fetching url: %v", err)
				return
			}
			if res.StatusCode() != 200 {
				log.WithField("status", res.StatusCode()).WithField("url", u).Warn("[bage] Error fetching url")

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
			if v, ok := b.kindStock[item.ProductName]; ok && v == item.Available { // 库存未变化, 不发送通知
				continue
			}
			b.kindStock[item.ProductName] = item.Available
			sendMsg = true
			body += fmt.Sprintf("%s: 库存 %d\n\n", item.ProductName, item.Available)
			body += fmt.Sprintf("购买链接: %s\n\n", item.BuyUrl)
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
	defer func() {
		CatchGoroutinePanic()
	}()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))

	if err != nil {
		log.Warn("[bage]Error parsing response: %v", err)
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
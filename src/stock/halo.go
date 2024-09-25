package stock

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"vps-stock/src/stock/db"
	"vps-stock/src/stock/vars"
)

type HaloVpsStockNotifier struct {
	vps vars.VPS
	bot BotNotifier
	cli *resty.Client
}

func NewHaloVpsStockNotifier(vps vars.VPS, bot BotNotifier) *HaloVpsStockNotifier {
	cli := resty.New().SetDebug(false)
	cli.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")
	cli.Header.Add("Referer", vps.BaseURL)
	return &HaloVpsStockNotifier{
		vps: vps,
		bot: bot,
		cli: cli,
	}
}

func (b *HaloVpsStockNotifier) Notify() {
	if len(b.vps.Products) == 0 {
		return
	}
	if len(b.vps.BaseURL) == 0 {
		return
	}
	defer CatchGoroutinePanic()
	var wg sync.WaitGroup
	var items []*vars.VpsStockItem
	var mu sync.Mutex
	for _, product := range b.vps.Products {
		u := b.vps.ProductUrl + product.Name
		wg.Add(1)
		atomic.AddInt64(&TotalQuery, 1)
		product := product
		go func() {
			defer func() {
				wg.Done()
				CatchGoroutinePanic()
			}()

			res, err := b.cli.R().Get(u)
			log.WithField("url", u).Trace("[halo] fetching url")
			if err != nil {
				log.WithField("url", u).Warnf("[halo]Error fetching url: %v", err)
				return
			}
			if res.StatusCode() != 200 {
				log.WithField("status", res.StatusCode()).WithField("url", u).Warn("[halo]Error fetching url")
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
	var body = "📢 *Halo库存通知*\n\n"
	var sendMsg bool
	for _, item := range items {
		exists, _ := db.GetKindByKind(item.ProductName)
		if exists == nil {
			exists = &db.Kind{
				Kind: item.ProductName,
			}
		}
		if item.Available > 0 {
			if exists.Stock == item.Available {
				db.AddOrUpdateKind(exists)
				continue
			}
			exists.Stock = item.Available
			sendMsg = true
			body += fmt.Sprintf("%s: 库存 %d\n\n", item.ProductName, item.Available)
			body += fmt.Sprintf("购买链接: %s\n\n", item.BuyUrl)
		}
		db.AddOrUpdateKind(exists)
	}
	if sendMsg {
		b.bot.Notify(NotifyMessage{Text: body})
	}
}
func (b *HaloVpsStockNotifier) parseResponse(kind []string, body string) []*vars.VpsStockItem {

	defer func() {
		CatchGoroutinePanic()
	}()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))

	if err != nil {
		log.Warnf("[halo]Error parsing response: %v", err)
		return nil
	}

	var rtn []*vars.VpsStockItem

	doc.Find(".product").Each(func(i int, s *goquery.Selection) {

		spans := s.Find("header span")

		productName := spans.First().Text()
		available := s.Find("span.qty").Text()
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
		if available != "" {
			// 2. 获取 <em> 标签内 <span> 的值
			available = strings.Replace(available, "Available", "", -1)
			available = strings.TrimSpace(available) // 去掉多余的空格
			item.Available = cast.ToInt(available)
		}

		if item.Available == 0 {
			return
		}
		// 3. 获取购买链接
		buyUrl, _ := s.Find("a.btn-order-now").Attr("href")
		//[Markdown语法](https://markdown.com.cn)
		item.BuyUrl = b.vps.BaseURL + buyUrl
		item.BuyUrl = fmt.Sprintf("[%s](%s)", item.BuyUrl, item.BuyUrl)
		rtn = append(rtn, item)
	})
	return rtn
}

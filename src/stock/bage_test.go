package stock

import (
	"fmt"
	"testing"

	"github.com/bytedance/mockey"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"vps-stock/src/stock/db"
	"vps-stock/src/stock/vars"
)

func TestVerifyLastStock(t *testing.T) {
	type args struct {
		items             []*vars.VpsStockItem
		mockGetKindByKind func() (*db.Kind, error)
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestVerifyLastStock",
			args: args{
				items: []*vars.VpsStockItem{
					{
						ProductName: "测试商品1",
						Available:   1000,
						BuyUrl:      "https://www.bagevm.com/cart.php?a=add&pid=32",
					},
					{
						ProductName: "测试商品 2",
						Available:   100,
						BuyUrl:      "https://www.bagevm.com/cart.php?a=add&pid=32",
					},
				},
				mockGetKindByKind: func() (*db.Kind, error) {
					return &db.Kind{Stock: 9}, nil
				},
			},
		},
		{
			name: "TestVerifyLastStock",
			args: args{
				items: []*vars.VpsStockItem{
					{
						ProductName: "测试商品_1",
						Available:   0,
						BuyUrl:      "https://www.baidu.com",
					},
					{
						ProductName: "测试商品_2",
						Available:   0,
						BuyUrl:      "https://www.baidu.com",
					},
				},
				mockGetKindByKind: func() (*db.Kind, error) {
					return &db.Kind{Stock: 9}, nil
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockey.PatchConvey(tt.name, t, func() {
				mockey.Mock(db.GetKindByKind).Return(tt.args.mockGetKindByKind()).Build()
				mockey.Mock(db.AddOrUpdateKind).Return(nil).Build()
				got, got1 := VerifyLastStock(tt.args.items)
				if got {
					bot, err := tgbotapi.NewBotAPI("7727116717:AAH31RbD5ygRkuWGO1EaCcfKybAoirykxaY")
					got1 = fmt.Sprintf("📢 *BageVM 库存通知*\n\n%s", got1)

					var msg = tgbotapi.NewMessage(-1002398248297, got1)
					msg.ParseMode = tgbotapi.ModeMarkdownV2
					replacer := Replacer
					//msg.Text += "\n||spoiler||\n"
					//msg.Text += "`这是一段代码`\n"
					//msg.Text += "```\n这是另一段代码\n```\n"
					//msg.Text += "\n>这是一段引用\n"
					msg.Text = replacer.Replace(msg.Text)

					_, err = bot.Send(msg)
					fmt.Println(msg)
					if err != nil {
						fmt.Println(err.Error())
					}
				} else {
					fmt.Println("库存无变化")
				}
			})
		})
	}
}

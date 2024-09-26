package stock

import (
	"fmt"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-module/carbon/v2"
)

func TestHuMan(t *testing.T) {
	cStart := carbon.CreateFromStdTime(time.Now().Add(-time.Hour * 2))
	m := fmt.Sprintf("启动时间: %s\n运行时长: %s", cStart.Format(carbon.DateTimeFormat), cStart.DiffForHumans())
	fmt.Println(m)

	bot, err := tgbotapi.NewBotAPI("7727116717:AAH31RbD5ygRkuWGO1EaCcfKybAoirykxaY")
	if err != nil {
		fmt.Errorf("NewBotAPI error: %v", err)
	}

	s := `* BageVM: * 

> tw-hinet-vds

		Hinet Dynamic 600M



> hong-kong-servers

    Hong Kong - TINY , Hong Kong - SMALL

> los-angeles-servers

    Los Angeles - TINY

### singapore-servers

    - Singapore - SMALL

### japan-servers

    - Japan - SMALL

### united-kingdom-servers

    - United Kingdom - TINY



## Halo:

### tokyo-japan-bgp

    - Tokyo Japan BGP VPS-Traffic billing`

	//s = tgbotapi.EscapeText(tgbotapi.ModeMarkdown, s)
	//fmt.Println(s)
	var msg = tgbotapi.NewMessage(-1002398248297, s)
	msg.ParseMode = tgbotapi.ModeMarkdown
	_, err = bot.Send(msg)
	if err != nil {
		fmt.Println(err.Error())
	}

}

//
//func TestMarkdownV2(t *testing.T) {
//	bot, err := tgbotapi.NewBotAPI("7727116717:AAH31RbD5ygRkuWGO1EaCcfKybAoirykxaY")
//
//	s := tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, "*bold \\*text*\n_italic \\*text_\n__underline__\n~strikethrough~\n||spoiler||\n*bold _italic bold ~italic bold strikethrough ||italic bold strikethrough spoiler||~ __underline italic bold___ bold*\n[inline URL](http://www.example.com/)\n[inline mention of a user](tg://user?id=123456789)\n![👍](tg://emoji?id=5368324170671202286)\n`inline fixed-width code`\n```\n\tpre-formatted fixed-width code block\n\t```\n```python\n\tpre-formatted fixed-width code block written in the Python programming language\n\t```\n>Block quotation started\n>Block quotation continued\n>Block quotation continued\n>Block quotation continued\n>The last line of the block quotation\n**>The expandable block quotation started right after the previous block quotation\n>It is separated from the previous block quotation by an empty bold entity\n>Expandable block quotation continued\n>Hidden by default part of the expandable block quotation started\n>Expandable block quotation continued\n>The last line of the expandable block quotation with the expandability mark||\n**>")
//	s = "*bold \\*text*\n_italic \\*text_\n__underline__\n~strikethrough~\n||spoiler||\n*bold _italic bold ~italic bold strikethrough ||italic bold strikethrough spoiler||~ __underline italic bold___ bold*\n[inline URL](http://www.example.com/)\n[inline mention of a user](tg://user?id=123456789)\n![👍](tg://emoji?id=5368324170671202286)\n`inline fixed-width code`\n```\n\tpre-formatted fixed-width code block\n\t```\n```python\n\tpre-formatted fixed-width code block written in the Python programming language\n\t```\n>Block quotation started\n>Block quotation continued\n>Block quotation continued\n>Block quotation continued\n>The last line of the block quotation\n**>The expandable block quotation started right after the previous block quotation\n>It is separated from the previous block quotation by an empty bold entity\n>Expandable block quotation continued\n>Hidden by default part of the expandable block quotation started\n>Expandable block quotation continued\n>The last line of the expandable block quotation with the expandability mark||\n**>"
//	var msg = tgbotapi.NewMessage(-1002398248297, s)
//	msg.ParseMode = tgbotapi.ModeMarkdownV2
//	_, err = bot.Send(msg)
//	if err != nil {
//		fmt.Println(err.Error())
//	}
//
//}

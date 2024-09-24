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

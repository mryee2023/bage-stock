package stock

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang-module/carbon/v2"
)

func TestHuMan(t *testing.T) {
	cStart := carbon.CreateFromStdTime(time.Now().Add(-time.Hour * 2))
	m := fmt.Sprintf("启动时间: %s\n运行时长: %s", cStart.Format(carbon.DateTimeFormat), cStart.DiffForHumans())
	fmt.Println(m)
}

package stock

import (
	"encoding/json"
	"runtime"
)

func GetCallerName() string {
	pc, _, _, _ := runtime.Caller(2)
	return runtime.FuncForPC(pc).Name()
}

var TotalQuery int64

func ToJson(v interface{}) string {
	p, e := json.Marshal(v)
	if e != nil {
		return e.Error()
	}
	return string(p)
}

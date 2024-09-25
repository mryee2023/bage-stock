package vars

type VpsStockItem struct {
	ProductName string `json:"product_name"`
	Available   int    `json:"available"`
	BuyUrl      string `json:"buy_url"`
}

type VPS struct {
	Name       string    `yaml:"name"`
	BaseURL    string    `yaml:"baseUrl"`
	ProductUrl string    `yaml:"productUrl"`
	Products   []Product `yaml:"products"`
}
type Product struct {
	Name string   `yaml:"name"`
	Kind []string `yaml:"kind"`
}
type Notify struct {
	Platform string `yaml:"platform"`
	Key      string `yaml:"key"`
	ChatId   string `yaml:"chatId"`
}
type Config struct {
	VPS            []VPS  `yaml:"vps"`
	Notify         Notify `yaml:"notify"`
	CheckTime      string `yaml:"checkTime"`
	Frozen         string `yaml:"frozen"`
	Db             string `yaml:"db"`
	LogLevel       string `yaml:"logLevel"`
	AlterChannelId int64  `yaml:"alterChannelId"`
}

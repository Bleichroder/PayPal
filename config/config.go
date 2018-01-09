package config

type config struct {
	Port          int    `json:"port"`
	NsqAddr       string `json:"nsq_addr"`
	SqlAddr       string `json:"sqlAddr"`
	PayPalIPNUrl  string `json:"payPalIPNUrl"`
	PayPalBaseUrl string `json:""payPalBaseUrl`
	ClientID      string `json:"clientID"`
	Secret        string `json:"secret"`
	RedisHost     string `json:"redisHost"`
	RedisPassword string `json:"redisPassword"`
}

var Config = new(config)

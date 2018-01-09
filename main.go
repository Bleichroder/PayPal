package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"zxPayPal/beelog"
	"zxPayPal/config"
	"zxPayPal/db"
	"zxPayPal/paynsq"
	"zxPayPal/paypal"
)

var configFile = flag.String("cf", "./config/c.json", "json format config")

func main() {
	flag.Parse()
	cFile := *configFile
	beelog.InitLog()
	InitConfig(cFile)
	db.InitSql()
	paypal.RedisInit()
	paynsq.InitProducer(config.Config.NsqAddr)
	for i := 0; i < 3; i++ {
		paynsq.InitConsumer(config.Config.NsqAddr)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/pay/paypal/pay", paypal.Pay)
	mux.HandleFunc("/api/pay/paypal/IPN", paypal.PayPalIPN)
	mux.HandleFunc("/api/pay/paypal/Verify", paypal.PayPalVerify)
	mux.HandleFunc("/api/pay/paypal/userinfo", paypal.GetUserInfo)
	mux.HandleFunc("/api/pay/paypal/payout", paypal.Payout)

	server := &http.Server{Addr: fmt.Sprintf(":%d", config.Config.Port), Handler: mux}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	ServerErr := make(chan error, 1)
	go func() {

		select {
		case sig := <-sigChan:
			beelog.Log.Debug("中断信号")
			beelog.Log.Debug("%+v\n", sig)
			err := server.Shutdown(context.Background())
			ServerErr <- err
			beelog.Log.Debug("shuted down")
			paynsq.Stop()
			db.MysqlDB.Close()
		}
	}()
	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		if <-ServerErr != nil {
			beelog.Log.Error("http shutdown err:%s", err)
		} else {
			beelog.Log.Debug("http shutdown gracefully exit")
		}
	} else {
		beelog.Log.Error("http listenandserver err:%s", err)
	}
}

func InitConfig(f string) {
	content, err := ioutil.ReadFile(f)
	if err != nil {
		beelog.Log.Error("read config file failed:%s", err.Error())
	}
	err = json.Unmarshal(content, config.Config)
	if err != nil {
		beelog.Log.Error("umarshal config file faile:%s", err.Error())
	}
	beelog.Log.Debug("config:%+v", config.Config)
}

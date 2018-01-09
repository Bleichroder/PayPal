package paynsq

import (
	"encoding/json"
	"time"
	"zxPayPal/beelog"
	"zxPayPal/db"
	"zxPayPal/paypal/model"

	"github.com/nsqio/go-nsq"
)

type PayPalConsumer struct{}

func (*PayPalConsumer) HandleMessage(msg *nsq.Message) error {
	beelog.Log.Debug("香蕉处理完毕，receive", msg.NSQDAddress, "message:", string(msg.Body))

	paylist := new(model.PayPalList)
	err := json.Unmarshal(msg.Body, paylist)
	if err != nil {
		beelog.Log.Error("consumer unmarshal message failed, %s", err.Error())
		return err
	}
	if msg.Attempts <= 3 {
		err = UpdatePay(paylist)
		if err != nil {
			beelog.Log.Error("update pay failed, %s", err.Error())
			msg.RequeueWithoutBackoff(time.Second * 3)
			return err
		}
	} else {
		beelog.Log.Error("尝试3次，放弃更新PayList %+v", paylist)
	}

	return nil
}

func UpdatePay(paylist *model.PayPalList) error {
	result, err := db.Exec("update paypal_pay set status = ?, updated = ? where id = ? and status != 3", paylist.Status, time.Now().Format("2006-01-02 15:04:05"), paylist.Id)
	if err != nil {
		beelog.Log.Error("database err:%s", err.Error())
		return err
	}
	if ra, _ := result.RowsAffected(); ra == 0 {
		beelog.Log.Error("affected rows is 0")
	}
	return nil
}

package paynsq

import (
	"fmt"
	"time"
	"zxPayPal/beelog"

	"github.com/nsqio/go-nsq"
)

var nsqProducer *nsq.Producer
var nsqConsumers []*nsq.Consumer

//初始化生产者
func InitProducer(nsqAddr string) {
	var err error
	nsqProducer, err = nsq.NewProducer(nsqAddr, nsq.NewConfig())
	if err != nil {
		beelog.Log.Error(fmt.Sprintf("new nsq producer error:%s", err))
		return
	}
}

//发布消息
func Publish(topic string, message []byte) error {
	var err error
	if nsqProducer != nil {
		if len(message) == 0 { //不能发布空串，否则会导致error
			return nil
		}
		err = nsqProducer.Publish(topic, message) // 发布消息
		return err
	}
	return fmt.Errorf("producer is nil", err)
}

//初始化消费者
func InitConsumer(nsqAddr string) {
	var topic, channel string
	topic = fmt.Sprintf("update_status")
	channel = "pay"
	cfg := nsq.NewConfig()
	cfg.LookupdPollInterval = time.Second          //设置重连时间
	c, err := nsq.NewConsumer(topic, channel, cfg) // 新建一个消费者
	if err != nil {
		beelog.Log.Error(fmt.Sprintf("new nsq consumer error:%s", err))
	}
	c.SetLogger(nil, 0)             //屏蔽系统日志
	c.AddHandler(&PayPalConsumer{}) // 添加消费者接口

	//建立NSQLookupd连接
	if err := c.ConnectToNSQD(nsqAddr); err != nil {
		beelog.Log.Error(fmt.Sprintf("new nsq consumer connect error:%s", err))
		return
	}
	nsqConsumers = append(nsqConsumers, c)
}

func Stop() {
	for _, c := range nsqConsumers {
		c.Stop()
	}
	nsqProducer.Stop()
}

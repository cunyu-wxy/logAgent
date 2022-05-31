package kafka

import (
	"fmt"
	"github.com/Shopify/sarama"
	"time"
)

//专门往kafka写日志的模块

type logData struct {
	topic string
	data  string
}

var (
	client      sarama.SyncProducer //声明一个全局连接kafka的生产者client
	logDataChan chan *logData
)

//Init 初始化client
func Init(addrs []string, maxSize int) (err error) {
	config := sarama.NewConfig()
	//toil包使用
	config.Producer.RequiredAcks = sarama.WaitForAll          //发送完数据需要leader和follow都确定
	config.Producer.Partitioner = sarama.NewRandomPartitioner //新选出一个portition
	config.Producer.Return.Successes = true                   //成功交付的信息将在success Chanel返回

	//连接kafka
	client, err = sarama.NewSyncProducer(addrs, config)
	if err != nil {
		fmt.Println("producer closed,err:", err)
		return
	}
	//初始化logDataChan
	logDataChan = make(chan *logData, maxSize)
	//开启后台goroutine从通道中取数据发往Kafka
	go sendToKafka()
	return
}

//外部开放函数，该函数只把日志数据发送到内部的channel中

func SendToChan(topic, data string) {
	msg := &logData{
		topic: topic,
		data:  data,
	}
	logDataChan <- msg
}

//发送日志到Kafka的函数
func sendToKafka() {
	for {
		select {
		case ld := <-logDataChan:
			//构造一个信息
			msg := &sarama.ProducerMessage{}
			msg.Topic = ld.topic
			msg.Value = sarama.StringEncoder(ld.data)
			//发送到Kafka
			pid, offset, err := client.SendMessage(msg)
			if err != nil {
				fmt.Println("send message failed,err:", err)
				return
			}
			fmt.Printf("pid:%v offset:%v\n\n", pid, offset)
		default:
			time.Sleep(time.Millisecond * 50)
		}
	}
}

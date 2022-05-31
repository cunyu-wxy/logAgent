package taillog

import (
	"context"
	"fmt"
	"github.com/hpcloud/tail"
	"logAgent/kafka"
)

//专门负责从日志文件收集日志的模块

//TailTask: 一个日志收集的任务

type TailTask struct {
	path     string
	topic    string
	instance *tail.Tail
	//为了退出t.run()
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewTailTask(path, topic string) (tailObj *TailTask) {
	ctx, cancel := context.WithCancel(context.Background())
	tailObj = &TailTask{
		path:       path,
		topic:      topic,
		ctx:        ctx,
		cancelFunc: cancel,
	}
	tailObj.init() //根据路径去打开对应日志
	return
}

func (t *TailTask) init() {
	config := tail.Config{
		ReOpen:    true,                                 //重新打开
		Follow:    true,                                 //是否跟随
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, //从文件的哪个位置开始读
		MustExist: false,                                //文件不存在不报错
		Poll:      true,
	}
	var err error
	t.instance, err = tail.TailFile(t.path, config)
	if err != nil {
		fmt.Println("tail file failed,err:", err)
	}
	//goroutine随主函数退出
	go t.run() //直接去采集日志发送到Kafka
}

func (t *TailTask) run() {
	for {
		select {
		case <-t.ctx.Done():
			fmt.Printf("tail task:%s_%s 结束了...\n", t.path, t.topic)
			return
		case line := <-t.instance.Lines:
			//kafka.SendToKafka(t.topic, line.Text) //函数调用函数（数据量较小时可用，无所谓）
			//将日志发送到通道中
			kafka.SendToChan(t.topic, line.Text)
			//使用Kafka包中单独的goroutine去取日志发送到Kafka
		}
	}
}

//func ReadLog() {
//	var (
//		line *tail.Line
//		ok   bool
//	)
//	for {
//		line, ok = <-tailObj.Lines
//		if !ok {
//			fmt.Printf("tail file closs reopen, filename:%s\n", tailObj.Filename)
//			time.Sleep(time.Second)
//			continue
//		}
//		fmt.Println("line:", line.Text)
//	}
//}

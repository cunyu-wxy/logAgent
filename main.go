package main

import (
	"fmt"
	"gopkg.in/ini.v1"
	"logAgent/conf"
	"logAgent/etcd"
	"logAgent/kafka"
	"logAgent/taillog"
	"logAgent/utils"
	"sync"
	"time"
)

//logAgent入口组件

var (
	cfg = new(conf.AppConf)
)

//func run() {
//	//读取日志
//	for {
//		select {
//		case line := <-taillog.ReadChan():
//			//发送到Kafka
//			kafka.SendToKafka(cfg.KafkaConf.Topic, line.Text)
//		default:
//			time.Sleep(time.Second)
//		}
//	}
//}

func main() {
	//加载配置文件
	err := ini.MapTo(cfg, "./conf/config.ini")
	if err != nil {
		fmt.Printf("load ini failed,err:%v\n", err)
	}
	//初始化kafka连接
	err = kafka.Init([]string{cfg.KafkaConf.Address}, cfg.KafkaConf.ChanMaxSize)
	if err != nil {
		fmt.Printf("Init kafka failed,err:%v\n", err)
		return
	}
	fmt.Println("init kafka success.")
	//初始化Etcd
	err = etcd.Init(cfg.EtcdConf.Address, time.Duration(cfg.EtcdConf.Timeout)*time.Second)
	if err != nil {
		fmt.Printf("init etcd failed,err:%v\n", err)
		return
	}
	fmt.Println("init etcd success.")
	//实现每个logAgent都拉取自己的配置，根据自身ip区分
	ipstr, err := utils.GetOutBoundIP()
	if err != nil {
		panic(err)
	}
	etcdConfKey := fmt.Sprintf(cfg.EtcdConf.Key, ipstr)
	// 从etcd中获取日志收集项的配置信息
	logEntryConf, err := etcd.GetConf(etcdConfKey)
	if err != nil {
		fmt.Printf("etcd.GetConf failed,err:%v\n", err)
		return
	}
	fmt.Printf("get conf from etcd success,%v\n", logEntryConf)

	for i, v := range logEntryConf {
		fmt.Printf("index:%v value:%v\n", i, v)
	}
	//收集日志发往Kafka
	taillog.Init(logEntryConf)
	//因为NewConfChan访问了tskMgr的newConfChan，这个channel是在taillog.Init(logEntryConf)被执行的
	// 派一个哨兵去监视日志收集项的变化，有变化及时通知logAgent实现热加载
	newConfChan := taillog.NewConfChan() //从taillog包中获取对外暴露的通道
	var wg sync.WaitGroup
	wg.Add(1)
	go etcd.WatchConf(etcdConfKey, newConfChan) //哨兵发现配置更新通知上面的通道
	wg.Wait()
	//具体业务
	//run()
}

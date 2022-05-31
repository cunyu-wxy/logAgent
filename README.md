# logAgent-自适应配置日志收集

# 项目背景
每个业务系统都有日志，当系统出现问题时，需要通过日志信息来定位和解决问题，当系统机器较少时，登录到服务器上查看即可定位，但是当下云原生时代，系统机器规模巨大，登录到具体机器上查看几乎不现实

# 系统功能特点
- 基于golang开发
- 可自动获取机器配置文件，根据配置文件自适应记录日志
- 实时收集日志存储到中心系统
- 对日志建立索引，可通过ES快速检索
- web界面实现日志展示与检索

# 面对的问题
实时日志量非常大，每天日志量亿级别，需要日志准时收集，延迟控制分钟级，系统架构可以支持水平扩展

# 业界方案
__ELK__

![ELK](https://pica.zhimg.com/v2-b349e241dfc66008f0b911bd4d761e5c_1440w.jpg?source=172ae18b)

__ELK的问题__
- 运维成本高，每增加一个日志收集项，都要手动修改配置
- 监控缺失，无法准确获取logStash(Kafka)状态信息
- 无法定制开发维护

# 架构
每台机器通过Etcd记录配置日志项，用户通过配置Etcd告知需要记录的日志项，logAgent模块通过读取Etcd配置项来得知需要记录的日志，当配置更新时，logAgent自动得知配置更改，拉取需要的日志信息，并发送到Kafka中，Kafka记录保存日志，供用户消费或持久化。

# 实现步骤
```go
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
```

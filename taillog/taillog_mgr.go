package taillog

import (
	"fmt"
	"logAgent/etcd"
	"time"
)

var tskMgr *tailLogMgr

// tailTask manager
type tailLogMgr struct {
	logEntry    []*etcd.LogEntry
	tskMap      map[string]*TailTask
	newConfChan chan []*etcd.LogEntry
}

func Init(logEntryConf []*etcd.LogEntry) {
	tskMgr = &tailLogMgr{
		logEntry:    logEntryConf, //保存当前日志收集项信息
		tskMap:      make(map[string]*TailTask, 16),
		newConfChan: make(chan []*etcd.LogEntry), //无缓冲区的通道
	}
	for _, logEntry := range logEntryConf {
		//conf: *etcd.LogEntry
		//logEntry: 要收集的文件的路径
		//记录初始化的时候起了多少个tailTask，便于后续判断
		tailObj := NewTailTask(logEntry.Path, logEntry.Topic)
		mk := fmt.Sprintf("%s_%s", logEntry.Path, logEntry.Topic)
		tskMgr.tskMap[mk] = tailObj
	}
	go tskMgr.run()
}

//监听自己的NewConfChan,有新的配置过来就做对应的处理

func (t *tailLogMgr) run() {
	for {
		select {
		case newConf := <-t.newConfChan:
			for _, conf := range newConf {
				mk := fmt.Sprintf("%s_%s", conf.Path, conf.Topic)
				_, ok := t.tskMap[mk]
				if ok {
					//已经有了，不做操作
					continue
				} else {
					tailObj := NewTailTask(conf.Path, conf.Topic)
					t.tskMap[mk] = tailObj
				}
			}
			//将原t.tskMap有，但newConf没有的删掉
			for _, c1 := range t.logEntry {
				isDelete := true
				for _, c2 := range newConf {
					if c2.Path == c1.Path && c2.Topic == c1.Topic {
						isDelete = false
						continue
					}
				}
				if isDelete {
					//把c1对应的tailObj停掉
					mk := fmt.Sprintf("%s_%s", c1.Path, c1.Topic)
					t.tskMap[mk].cancelFunc()
				}
			}
			//配置变更
			fmt.Println("检测到新的配置：", newConf)
		default:
			time.Sleep(time.Second)
		}
	}
}

// NewConfChan 暴露tskMgr的newConChan
func NewConfChan() chan<- []*etcd.LogEntry {
	return tskMgr.newConfChan
}

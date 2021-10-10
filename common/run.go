package common

import "github.com/sirupsen/logrus"

import (
	"sync"
	"time"
)

/**
* 初始化logrus配置
* @param		nil
* @return		nil
* func initLogConfig();
*/
func initLogConfig() {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:               true,
		FullTimestamp:             true,
		TimestampFormat:           "2006-01-02 15:04:05",
		DisableLevelTruncation:    true,
	})
	logrus.SetReportCaller(true)
}

var onceExecFunc sync.Once
var restartWaitTime = time.Duration(1)
//
/**
* 支持panic重启的函数运行
* @param		server func()	要执行的函数
* @return		nil
* func Run(server func());
*/
func Run(server func()) {
	onceExecFunc.Do(initLogConfig)
	resertWaitRun := sync.WaitGroup{}
	for {
		resertWaitRun.Add(1)
		go func() {
			defer resertWaitRun.Done()
			defer func() {
				for p := recover(); p != nil; p = recover() {
					logrus.Println("panic:", p)
				}
			}()
			server()
		}()
		resertWaitRun.Wait()
		time.Sleep(restartWaitTime * time.Second)
	}
}

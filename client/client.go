package main

import (
	"github.com/sirupsen/logrus"
	"net"
	"safrp/client/config"
	"safrp/client/log"
	"safrp/common"
	"time"
)

func main() {
}

type safrpClient struct {
	connManage chan int		// 连接数控制器
	config *config.Config	// 配置
	log *logrus.Logger		// 日志
	ctx *common.Context		// 上下文
}

func (this *safrpClient) getPipe() {
	this.connManage <- 1	// 没有达到最大隧道数
}

func (this *safrpClient) putPipe() {
	<-this.connManage
}

func NewSafrpClient(conf *config.Config) *safrpClient {
	return &safrpClient{
		connManage: make(chan int, conf.PipeNum),
		config:conf,
		log:        log.GetLog(),
	}
}

func (this *safrpClient) getProtocol() string {
	return this.config.Protocol
}

func (this *safrpClient) getSafrpServerAddress() string {
	return this.config.SafrpServerIP + ":" + this.config.SafrpServerPort
}

func (this *safrpClient) connectSafrpServer() {
	this.getPipe()
	// 创建一个连接
	_, err := net.Dial(this.getProtocol(), this.getSafrpServerAddress())
	if err != nil {
		this.log.Errorln(err)
		time.Sleep(3 * time.Second)
		this.putPipe()
	}
}

func (this *safrpClient) Run() {
}


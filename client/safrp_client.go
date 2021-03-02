package main

import (
	"github.com/sirupsen/logrus"
	"net"
	"safrp/client/config"
	"safrp/client/context"
	"safrp/client/log"
	"time"
)

func NewSafrpClient(conf *config.Config, ctx *context.Context) *safrpClient {
	return &safrpClient{
		connManage: make(chan int, conf.PipeNum),
		config:conf,
		log:        log.GetLog(),
		ctx:ctx,
	}
}

type safrpClient struct {
	connManage chan int		// 连接数控制器
	config *config.Config	// 配置
	log *logrus.Logger		// 日志
	ctx *context.Context		// 上下文
}

func (this *safrpClient) getPipe() {
	this.connManage <- 1	// 没有达到最大隧道数
}

func (this *safrpClient) putPipe() {
	<-this.connManage
}

func (this *safrpClient) setCtxConn(conn *net.Conn, id uint64) {
	this.ctx.Conn[id] = conn
}

func (this *safrpClient) getCtxConn(id uint64) *net.Conn {
	return this.ctx.Conn[id]
}

func (this *safrpClient) Run() {
	for {
		this.getPipe()
		// 创建一个连接
		conn, err := net.Dial(this.getProtocol(), this.getSafrpServerAddress())
		if err != nil {
			this.log.Errorln(err)
			time.Sleep(3 * time.Second)
			this.putPipe()
		}
		// 获取连接编号
		id, _ := this.ctx.NumberPool.Get()
		this.setCtxConn(&conn, id)
		go this.runPipe(id)
	}
}

func (this *safrpClient) runPipe(id uint64) {

}

func (this *safrpClient) getProtocol() string {
	return this.config.Protocol
}

func (this *safrpClient) getSafrpServerAddress() string {
	return this.config.SafrpServerIP + ":" + this.config.SafrpServerPort
}

func (this *safrpClient) connectSafrpServer() {
}
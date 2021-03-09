package safrp_client

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"net"
	"safrp/client/config"
	"safrp/client/context"
	"safrp/client/log"
	"safrp/common"
	"strconv"
	"time"
)

func NewSafrpClient() *safrpClient {
	return &safrpClient{
		connManage: make(chan int, config.GetConfig().PipeNum),
		config:     config.GetConfig(),
		log:        log.GetLog(),
		ctx:        context.GetCtx(),
	}
}

type safrpClient struct {
	connManage chan int         // 连接数控制器
	config     *config.Config   // 配置
	log        *logrus.Logger   // 日志
	ctx        *context.Context // 上下文
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
		this.setCtxConn(conn, id)
		go this.runPipe(id)
	}
}

func (this *safrpClient) runPipe(id uint64) {
	defer func() {
		for r := recover(); r != nil; r = recover() {
			this.log.Println(r)
		}
		// 回收连接编号
		this.putPipe()
	}()
	// 请求连接safrp服务端
	this.connectSafrpServer(id)

	go this.PublishSafrpServerEvent(id)
	go this.SubscribeSafrpServerEvent(id)
	go this.PublishProxyEvent(id)
	go this.SubscribeProxyEvent(id)
	select {}
}

func (this *safrpClient) getProtocol() string {
	return this.config.Protocol
}

func (this *safrpClient) getSafrpServerAddress() string {
	return this.config.SafrpServerIP + ":" + this.config.SafrpServerPort
}

func (this *safrpClient) connectSafrpServer(id uint64) {
	this.ctx.Conn[id].Write([]byte(this.config.Password)) // 发送连接密码
	buf := make([]byte, 1)
	this.log.Printf("编号：%d，开始连接。。。\n", id)
	this.ctx.Conn[id].Read(buf) // 读取连接结果
	if buf[0] == '0' {
		panic("密码错误。。。")
	}
	this.log.Printf("编号：%d，连接成功。。。\n", id)
}

func (this *safrpClient) getPipe() {
	this.connManage <- 1 // 没有达到最大隧道数
}

func (this *safrpClient) putPipe() {
	<-this.connManage
}

func (this *safrpClient) setCtxConn(conn net.Conn, id uint64) {
	this.ctx.Conn[id] = conn
}

func (this *safrpClient) getCtxConn(id uint64) net.Conn {
	return this.ctx.Conn[id]
}

func (this *safrpClient) parese(data []byte) *common.DataPackage {
	var pack common.DataPackage
	ds := bytes.SplitN(data, []byte("\r\n"), 2)
	dsVal := bytes.Split(ds[0], []byte{' '})
	pack.Number, _ = strconv.Atoi(string(dsVal[0]))
	pack.Status = string(dsVal[2])
	pack.Data = ds[1][:len(ds[1])-7]
	return &pack
}
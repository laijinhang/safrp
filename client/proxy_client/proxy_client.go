package proxy_client

import (
	"net"
	"safrp/client/config"
	"safrp/client/context"
)

type proxyClient struct {
	ctx *context.Context		// 上下文
	conf *config.Config
}

 func NewproxyClient() *proxyClient {
	return &proxyClient{
		ctx:context.GetCtx(),
	}
}

func (this *proxyClient) Run() {
	go this.PublishProxyEvent()
	go this.SubscribeSafrpClientEvent()
	select {}
}

// 通过连接id获取一个连接
func (this *proxyClient) GetConnect(connId uint64) net.Conn {
	if this.ctx.Proxy[connId] == nil {
		this.ctx.Proxy[connId], _ = net.Dial("tcp", this.conf.HTTPIP + ":" + this.conf.HTTPPort)
	}
	return this.ctx.Proxy[connId]
}


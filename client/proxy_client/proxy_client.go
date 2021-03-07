package proxy_client

import (
	"net"
	"safrp/client/context"
)

type proxyClient struct {
	ctx *context.Context		// 上下文
}

 func NewproxyClient() *proxyClient {
	return &proxyClient{
		ctx:context.GetCtx(),
	}
}

func (this *proxyClient) Run() {

}

// 通过连接id获取一个连接
func (this *proxyClient) GetConnect(connId uint64) net.Conn {
	return nil
}


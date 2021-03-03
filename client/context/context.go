package context

import (
	"net"
	"safrp/client/config"
	"safrp/common"
)

type Context struct {
	NumberPool      *common.NumberPool
	FromSafrpServer []chan common.DataPackage
	ToSafrpServer   []chan common.DataPackage
	FromProxyServer chan common.DataPackage
	ToProxyServer   chan common.DataPackage

	ToClient []chan common.DataPackage
	ReadConn []*func()
	Conn     []net.Conn
}

var ctx Context

func init() {
	ctx = Context{
		NumberPool:      nil,
		FromSafrpServer: make([]chan common.DataPackage, config.GetConfig().PipeNum+1),
		ToSafrpServer:   make([]chan common.DataPackage, config.GetConfig().PipeNum+1),
		FromProxyServer: make(chan common.DataPackage, 3000),
		ToProxyServer:   make(chan common.DataPackage, 3000),
		ToClient:        make([]chan common.DataPackage, config.GetConfig().PipeNum+1),
		ReadConn:        make([]*func(), 3000),
		Conn:            make([]net.Conn, config.GetConfig().PipeNum+1),
	}
	for i := 0;i < config.GetConfig().PipeNum;i++ {
		ctx.FromSafrpServer[i] = make(chan common.DataPackage, 10)
		ctx.ToClient[i] = make(chan common.DataPackage, 10)
		ctx.ToSafrpServer[i] = make(chan common.DataPackage, 10)
	}
}

func GetCtx() *Context {
	return &ctx
}

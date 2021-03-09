package context

import (
	"net"
	"safrp/client/config"
	"safrp/common"
	"safrp/common/number_pool"
)

type Context struct {
	NumberPool      *number_pool.NumberPool
	FromSafrpServer []chan common.DataPackage
	ToSafrpServer   []chan common.DataPackage
	FromProxyServer chan common.DataPackage
	ToProxyServer   chan common.DataPackage

	ToClient []chan common.DataPackage
	ReadConn []*func()
	Conn     []net.Conn
	Proxy    []net.Conn
}

var ctx Context

func init() {
	ctx = Context{
		NumberPool:      number_pool.NewNumberPool(config.GetConfig().GetConnNum(), 1),
		FromSafrpServer: make([]chan common.DataPackage, config.GetConfig().PipeNum+1),
		ToSafrpServer:   make([]chan common.DataPackage, config.GetConfig().PipeNum+1),
		FromProxyServer: make(chan common.DataPackage, 3000),
		ToProxyServer:   make(chan common.DataPackage, 3000),
		ToClient:        make([]chan common.DataPackage, config.GetConfig().PipeNum+1),
		ReadConn:        make([]*func(), 3000),
		Conn:            make([]net.Conn, config.GetConfig().PipeNum+1),
		Proxy:           make([]net.Conn, config.GetConfig().ConnNum+1),
	}
	for i := 0; i < config.GetConfig().PipeNum; i++ {
		ctx.FromSafrpServer[i] = make(chan common.DataPackage, 10)
		ctx.ToClient[i] = make(chan common.DataPackage, 10)
		ctx.ToSafrpServer[i] = make(chan common.DataPackage, 10)
	}
}

func GetCtx() *Context {
	return &ctx
}

package context

import (
	"net"
	"safrp/common"
)

type Context struct {
	NumberPool *common.NumberPool
	FromSafrpServer []chan common.DataPackage
	ToSafrpServer   []chan common.DataPackage
	FromProxyServer chan common.DataPackage
	ToProxyServer   chan common.DataPackage

	ToClient []chan common.DataPackage
	ReadConn []*func()
	Conn []*net.Conn
}

var ctx Context

func init() {

}

func GetCtx() *Context {
	return &ctx
}
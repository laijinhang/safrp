package context

import (
	"net"
	"safrp/common"
)

type Context struct {
	FromSafrpServer []chan common.DataPackage
	ToSafrpServer   []chan common.DataPackage
	FromProxyServer chan common.DataPackage
	ToProxyServer   chan common.DataPackage

	ToClient []chan common.DataPackage
	ReadConn []*func()
	Conn []*net.Conn
}
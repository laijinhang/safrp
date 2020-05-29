package common

import (
	"net"
)

/*--------- 核心插件 -----------*/
// TCP连接插件
func TCPConnect(ctx *Context) {
	conn, err := net.Dial(ctx.Protocol, ctx.IP + ":" + ctx.Port)
	if err != nil {
		ctx.Log.Panicln(err)
		return
	}
	ctx.Conn = append(ctx.Conn.([]net.Conn), conn)
}

// TCP监听插件
func TCPListen(ctx *Context)  {
	conn, err := net.Listen(ctx.Protocol, ctx.IP + ":" + ctx.Port)
	if err != nil {
		ctx.Log.Panicln(err, ctx.Protocol, ctx.Protocol, ctx.IP + ":" + ctx.Port)
	}
	ctx.Conn = conn
}

// 连接安全验证插件
func checkConnectPassword(password string) bool {
	return false
}
// 判断心跳包插件
func ReadHeartbeat() {

}
// 与safrp客户端交互的数据解析插件
func ParsePackage(c net.Conn) {

}

/*-------------- 功能性插件 -----------------*/
// 限流插件
// IP记录插件
// 将记录写入mq中
// 将记录写到数据库
// 黑名单插件
// 白名单插件
// 插件接口
func plugInInterface(ctx *Context) {

}

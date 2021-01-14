package base

import "net"

func NewHTTP11(conn *net.Conn) *http11 {
	return &http11{
		conn:      conn,
		HeaderMap: make(map[string]string),
		Header:    "",
		Data:      "",
	}
}

/*
对于一次完整的HTTP过程：
1. 传输了一部分，TCP连接就断了 => HTTP结束
2. 错误的HTTP数据请求 => 服务端会一直等待，直到这条TCP连接断开
3. 完整的HTTP数据请求 => 请求成功
*/
type http11 struct {
	conn      *net.Conn
	HeaderMap map[string]string
	Header    string // HTTP请求头
	Data      string // HTTP数据部分
}

/*
一直读取，直到结束，结束的条件只会有两种：
1. 要么这条TCP连接断掉了
2. 要么数据传输完成
*/
func (this *http11) Parse() string {
	// 1、解析HTTP请求头

	// 2、解析HTTP数据部分
	return ""
}

package main

import "net"

// 数据中心
type DataCenter struct {
	conn []net.Conn
}

// 发送数据给代理客户端
func (this *DataCenter) SendDataToProxyClient() {

}

// 发送数据给safrp客户端
func (this *DataCenter) SendDataToSafrpClient() {
	select {

	}
}

func (this *DataCenter) RegisterPipe(conn net.Conn, pipeNumber uint64) {
	this.conn[pipeNumber] = conn
}
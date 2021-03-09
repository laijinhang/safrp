package safrp_client

import "fmt"

// 发布 向safrp服务端写数据 事件
func (this *safrpClient) PublishSafrpServerEvent(id uint64) {
	//buf := make([]byte, 1024)
	//this.ctx.Conn[id].Read(buf)
	//this.ctx.FromSafrpServer[id] <- nil
}

// 订阅 从safrp服务端读数据 事件
func (this *safrpClient) SubscribeSafrpServerEvent(id uint64) {
	buf := make([]byte, 1024)
	for {
		num, err := this.ctx.Conn[id].Read(buf)
		if err != nil {
			return
		}
		p := this.parese(buf[:num])
		fmt.Println(p.Number, p.Status, string(p.Data))
		this.ctx.ToProxyServer <- *p
	}
}

// 发布 向代理写数据 事件
func (this *safrpClient) PublishProxyEvent(id uint64) {
}

// 订阅 从代理读数据 事件
func (this *safrpClient) SubscribeProxyEvent(id uint64) {

}

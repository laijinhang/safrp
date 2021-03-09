package proxy_client

// 发布 代理 事件
func (this *proxyClient) PublishProxyEvent() {

}

// 订阅 从safrp客户端读数据 事件
func (this *proxyClient) SubscribeSafrpClientEvent() {
	for {
		select {
		case pack := <- this.ctx.ToProxyServer:
			conn := this.GetConnect(uint64(pack.Number))
			conn.Write(pack.Data)
		}
	}
}
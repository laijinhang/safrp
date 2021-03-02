package main

import "safrp/client/context"

type proxyClient struct {
	ctx *context.Context		// 上下文
}

func NewproxyClient(ctx *context.Context) *proxyClient {
	return &proxyClient{
		ctx:ctx,
	}
}

func (this *proxyClient) Run() {

}
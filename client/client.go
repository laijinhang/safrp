package main

import (
	"safrp/client/config"
	"safrp/client/context"
)

func main() {
	go RunSafrpClient()
	go RunProxyClient()
	select {}
}

func RunSafrpClient() {
	safrpClient := NewSafrpClient(config.GetConfig(), context.GetCtx())
	safrpClient.Run()
}

func RunProxyClient() {
	proxyClient := NewproxyClient(context.GetCtx())
	proxyClient.Run()
}

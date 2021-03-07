package main

import (
	"fmt"
	"os"
	"os/signal"
	"safrp/client/proxy_client"
	"safrp/client/safrp_client"
)

func main() {
	go RunSafrpClient()
	go RunProxyClient()
	go Exit()
	select {}
}

func RunSafrpClient() {
	safrpClient := safrp_client.NewSafrpClient()
	safrpClient.Run()
}

func RunProxyClient() {
	proxyClient := proxy_client.NewproxyClient()
	proxyClient.Run()
}

func Exit() {
	c := make(chan os.Signal)
	//监听所有信号
	signal.Notify(c)
	//阻塞直到有信号传入
	fmt.Println("启动")
	s := <-c
	fmt.Println("退出", s)
	os.Exit(1)
}

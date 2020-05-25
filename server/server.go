package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"safrp/common"
	"sync"
)


var confs []common.Config

const (
	TCPStreamLength = 1024 * 8	// 1个TCP窗口大小
	UDPDatagramLength = 1472	// 1个UDP报文长度
)

func init() {
	cfg, err := ini.Load("./safrp.ini")
	if err != nil {
		logrus.Fatal("Fail to read file: ", err)
	}

	for i := 1; len(cfg.Section(fmt.Sprintf("pipe%d", i)).Keys()) != 0;i++ {
		pipeConf := cfg.Section(fmt.Sprintf("pipe%d", i))
		confs = append(confs, common.Config{
			IP:           pipeConf.Key("ip").String(),
			ExtranetPort: pipeConf.Key("extranet_port").String(),
			ServerPort:   pipeConf.Key("server_port").String(),
			Proxy:        pipeConf.Key("proxy").String(),
			PipeNum: func(t uint, err error) uint8 {
				return uint8(t)
			}(pipeConf.Key("pipe_num").Uint()),
		})
	}
}

func main() {
	common.Run(func() {
		// 启动 pipe组件
		for i := 0;i < len(confs);i++ {
			go func() {
				ctx := common.Context{
					Conf:confs[i],
					NumberPool:common.NewNumberPool(3000, 1),
					SendData:make(chan common.DataPackage, 1000),
					ReadDate:make(chan common.DataPackage, 1000),
				}

				es := UnitFactory(ctx.Conf.Proxy, ctx.Conf.IP, ctx.Conf.ExtranetPort)
				ss := UnitFactory(ctx.Conf.Proxy, ctx.Conf.IP, ctx.Conf.ServerPort)

				extranetServer.Register(&ctx, &es)
				safrpServer.Register(&ctx, &ss)

				go common.Run(func() {
					// 对外
					ExtranetServer(&ctx)
				})
				go common.Run(func() {
					// 对safrp客户端
					SafrpServer(&ctx)
				})
			}()
		}
		select {}
	})
}

func ExtranetServer(ctx *common.Context) {
	go extranetServer.Get(ctx).ReadServer(nil)
	extranetServer.Get(ctx).SendServer(ctx.SendData)
}

func SafrpServer(ctx *common.Context) {
	go safrpServer.Get(ctx).ReadServer(nil)
	safrpServer.Get(ctx).SendServer(ctx.ReadDate)
}

// 单例模式
var extranetServer single
var safrpServer single

type single struct {
	lock sync.Locker
	server map[*common.Context]common.Server
}

func (s *single)Register(ctx *common.Context, server *common.Server) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.server[ctx] = *server
}

func (s *single)Get(ctx *common.Context) common.Server {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.server[ctx]
}

// 组件工厂
func UnitFactory(proxy, ip, port string) common.Server {
	switch proxy {
	case "tcp":
		return &common.TCPServer{IP:ip, Port:port}
	case "udp":
		return &common.UDPServer{IP:ip, Port:port}
	case "http":
		return &common.HTTPServer{IP: ip, Port:port}
	}
	return nil
}

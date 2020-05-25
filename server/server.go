package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"net"
	"safrp/common"
	"sync"
	"time"
)

type Config struct {
	IP           string
	ExtranetPort string
	ServerPort   string
	Proxy        string
	PipeNum      uint8
}

type Context struct {
	Conf Config
	NumberPool *common.NumberPool
	ReadDate chan DataPackage
	SendData chan DataPackage
	DateLength int
}

type DataPackage struct {
	Number int
	Data []byte
}

var confs []Config

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
		confs = append(confs, Config{
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
				ctx := Context{
					Conf:confs[i],
					NumberPool:common.NewNumberPool(3000, 1),
					SendData:make(chan DataPackage, 1000),
					ReadDate:make(chan DataPackage, 1000),
				}

				es := UnitFactory(ctx.Conf.Proxy, ctx.Conf.ExtranetPort)
				ss := UnitFactory(ctx.Conf.Proxy, ctx.Conf.ServerPort)

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

func ExtranetServer(ctx *Context) {
	go extranetServer.Get(ctx).Read(nil)
	extranetServer.Get(ctx).Send(ctx.SendData)
}

func SafrpServer(ctx *Context) {
	go safrpServer.Get(ctx).Read(ctx.ReadDate)
	safrpServer.Get(ctx).Send(ctx.ReadDate)
}

// 单例模式
var extranetServer single
var safrpServer single

type single struct {
	lock sync.Locker
	server map[*Context]Server
}

func (s *single)Register(ctx *Context, server *Server) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.server[ctx] = *server
}

func (s *single)Get(ctx *Context) Server {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.server[ctx]
}

// 组件工厂
func UnitFactory(proxy, port string) Server {
	switch proxy {
	case "tcp":
		return &TCPServer{Port:port}
	case "udp":
		return &UDPServer{Port:port}
	case "http":
		return &HTTPServer{Port:port}
	}
	return nil
}

type Server interface {
	Read(interface{})
	Send(interface{})
	Proxy() string
}

type TCPServer struct {
	Port string
}

func (t *TCPServer) Read(c interface{}) {
	listen, err := net.Listen("tcp", conf.ServerIP + ":" + conf.ServerTCPPort)
	logrus.Infoln("listen :" + conf.ServerIP + ":" + conf.ServerTCPPort + " ...")
	if err != nil {
		panic(err)
	}
	for {
		client, err := listen.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer func() {
				for p := recover();p != nil;p = recover(){
					logrus.Errorln(p)
				}
			}()
			defer c.Close()
			num := -1
			for c := 0;num == -1;c++ {
				tNum, ok := ConnPool.Get()
				if ok {
					num = int(tNum)
					break
				} else {
					time.Sleep(50 * time.Millisecond)
				}
				if c == 20 {
					return
				}
			}

			tcpFromClientStream[num] = make(chan TCPData, 30)
			logrus.Infoln("请求：", client.RemoteAddr(), num)
			defer func() {
				tcpToClientStream <- TCPData{
					ConnId: num,
					Data:   []byte(""),
				}
				ConnPool.Put(num)
				c.Close()
				close(tcpFromClientStream[num].(chan TCPData))
			}()
			go ExtranetTCPRead(c, num)
			ExtranetTCPSend(c, num)
		}(client)
	}
}

func (t *TCPServer) Send(c interface{}) {}

func (t *TCPServer) Proxy() string {
	return "tcp"
}

type UDPServer struct {
	Port string
}

func (u *UDPServer) Read(c interface{}) {}
func (u *UDPServer) Send(c interface{}) {}
func (u *UDPServer) Proxy() string {
	return "udp"
}

type HTTPServer struct {
	Port string
}

func (h *HTTPServer) Read(c interface{}) {}
func (h *HTTPServer) Send(c interface{}) {}
func (h *HTTPServer) Proxy() string {
	return "http"
}
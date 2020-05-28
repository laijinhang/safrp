package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"io"
	"net"
	"safrp/common"
	"sync"
	"time"
)


type Config struct {
	IP           string
	ExtranetPort string
	ServerPort   string
	Protocol     string
	PipeNum      uint8
}

type Context struct {
	ConnManage []net.Conn			// 管理与safrp之间的连接
	ConnClose []chan bool
	ConnNumberPool *common.NumberPool	// 连接编号池
	ConnDataChan []chan common.DataPackage
	PipeConnControllor chan int		// 限制safrp客户端与safrp服务端之间的连接数
}

var confs []Config
var BufSize = 1024 * 10 * 8
var BufPool = sync.Pool{New: func() interface{} {return make([]byte, BufSize)}}

const (
	TCPStreamLength = 1024 * 8	// 1个TCP窗口大小
	UDPDatagramLength = 1472	// 1个UDP报文长度
)

var TCPDataEnd = []byte{'<','e','>'}

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
			Protocol:     pipeConf.Key("protocol").String(),
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
			go func(i int) {
				log := logrus.New()
				log.SetLevel(logrus.TraceLevel)
				log.SetFormatter(&logrus.TextFormatter{
					ForceColors:               true,
					FullTimestamp:             true,
					TimestampFormat:           "2006-01-02 15:04:05",
					DisableSorting:            false,
					SortingFunc:               nil,
					DisableLevelTruncation:    true,
					QuoteEmptyFields:          false,
					FieldMap:                  nil,
					CallerPrettyfier:          nil,
				})
				log.SetReportCaller(true)
				log.Printf("启动pipe%d\n", i + 1)
				ctx := common.Context{
					Conf:confs[i],
					NumberPool:common.NewNumberPool(3000, 1),
					SendData:make(chan common.DataPackage, 1000),
					ReadDate:make(chan common.DataPackage, 1000),
					Log: log,
				}

				es := UnitFactory(ctx.Conf.(Config).Protocol, ctx.Conf.(Config).IP, ctx.Conf.(Config).ExtranetPort)
				ss := UnitFactory(ctx.Conf.(Config).Protocol, ctx.Conf.(Config).IP, ctx.Conf.(Config).ServerPort)


				connDataChan := make([]chan common.DataPackage, ctx.Conf.(Config).PipeNum+1)
				ctx1 := common.Context{
					Conf:       confs[i],
					NumberPool: common.NewNumberPool(3000, 1),
					ReadDate:   ctx.ReadDate,
					SendData:   ctx.SendData,

					IP:         "",
					Port:       "",
					Protocol:   GetBaseProtocol(confs[i].Protocol),
					Conn:       nil,
					Log:        ctx.Log,
					Expand:     Context{
						ConnManage:         make([]net.Conn, ctx.Conf.(Config).PipeNum+1),
						PipeConnControllor: make(chan int, ctx.Conf.(Config).PipeNum),
						ConnNumberPool: common.NewNumberPool(uint64(ctx.Conf.(Config).PipeNum), uint64(1)),
						ConnDataChan: connDataChan,
					},
				}
				ctx2 := common.Context{
					Conf:       confs[i],
					NumberPool: common.NewNumberPool(3000, 1),
					ReadDate:   ctx.ReadDate,
					SendData:   ctx.SendData,

					IP:         "",
					Port:       "",
					Protocol:   GetBaseProtocol(confs[i].Protocol),
					Conn:       nil,
					Log:        ctx.Log,
					Expand:     Context{
						ConnManage:         make([]net.Conn, ctx.Conf.(Config).PipeNum+1),
						ConnClose:			make([]chan bool, ctx.Conf.(Config).PipeNum+1),
						PipeConnControllor: make(chan int, ctx.Conf.(Config).PipeNum),
						ConnNumberPool: common.NewNumberPool(uint64(ctx.Conf.(Config).PipeNum), uint64(1)),
					},
				}

				extranetServer.Register(&ctx1, &es)
				safrpServer.Register(&ctx2, &ss)


				go common.Run(func() {
					// 对外
					ExtranetServer(&ctx1)
				})
				go common.Run(func() {
					// 对safrp客户端
					SafrpServer(&ctx2)
				})
			}(i)
		}
		select {}
	})
}

func ExtranetServer(ctx *common.Context) {
	extranetServer.Get(ctx).Server(ctx, []func(ctx *common.Context) {
		TCPListen,
		ExtranetTCPServer,
	})
}

func SafrpServer(ctx *common.Context) {
	safrpServer.Get(ctx).Server(ctx, []func(ctx *common.Context) {
		TCPListen,
	})
	go safrpServer.Get(ctx).Server(ctx, []func(ctx *common.Context) {
		SafrpTCPServer,
	})
	go safrpServer.Get(ctx).ReadServer(ctx, []func(ctx *common.Context) {
		SafrpTCPSend,
	})
	safrpServer.Get(ctx).SendServer(ctx, []func(ctx *common.Context) {
		SafrpTCPRead,
	})
}

// 单例模式
var extranetServer = NewSingle()
var safrpServer = NewSingle()

type single struct {
	lock sync.Mutex
	server map[*common.Context]common.Server
}

func NewSingle() single {
	return single{
		server: make(map[*common.Context]common.Server),
	}
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
func UnitFactory(protocol, ip, port string) common.Server {
	switch protocol {
	case "tcp":
		return &common.TCPServer{IP:ip, Port:port}
	case "udp":
		return &common.UDPServer{IP:ip, Port:port}
	case "http":
		return &common.HTTPServer{IP: ip, Port:port}
	}
	return nil
}

func GetBaseProtocol(protocol string) string {
	switch protocol {
	case "tcp", "http":
		return "tcp"
	case "udp":
		return "udp"
	}
	return ""
}

/*--------- 核心插件 -----------*/
// TCP连接插件
func TCPConnect(ctx *common.Context) {
	conn, err := net.Dial(ctx.Protocol, ctx.IP + ":" + ctx.Port)
	if err != nil {
		ctx.Log.Panicln(err)
	}
	ctx.Conn = conn
}

// TCP监听插件
func TCPListen(ctx *common.Context)  {
	conn, err := net.Listen(ctx.Protocol, ctx.IP + ":" + ctx.Port)
	if err != nil {
		ctx.Log.Panicln(err, ctx.Protocol, ctx.Protocol, ctx.IP + ":" + ctx.Port)
	}
	ctx.Conn = conn
}

// safrp TCP对外服务插件
func ExtranetTCPServer(ctx *common.Context) {
	for {
		client, err := ctx.Conn.(net.Listener).Accept()
		if err != nil {
			continue
		}
		go func(client net.Conn) {
			defer client.Close()

			num := -1
			for c := 0;num == -1;c++ {
				tNum, ok := ctx.NumberPool.Get()
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

			connClose := make(chan bool, 2)
			// 读
			go func() {
				buf := BufPool.Get().([]byte)
				defer func() {
					BufPool.Put(buf)
					connClose <- true
				}()
				for {
					select {
					case <-connClose:
						return
					default:
						// 一直读，直到连接关闭
						err := ctx.Conn.(net.Conn).SetReadDeadline(time.Now().Add(1 * time.Second))
						if err != nil {
							return
						}
						n, err := ctx.Conn.(net.Conn).Read(buf)
						if err != nil {
							if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
								continue
							}
							return
						}
						ctx.ReadDate <- common.DataPackage{
							Number: num,
							Data: buf[:n]}
					}
				}
			}()
			// 写
			func () {
				defer func() {
					connClose <- true
				}()
				for {
					select {
					case <-connClose:
						return
					case data := <-ctx.Expand.(Context).ConnDataChan[num]:
						// 有数据来的话，就一直写，直到连接关闭
						err := client.SetWriteDeadline(time.Now().Add(time.Second))
						if err != nil {
							ctx.Log.Errorln(err)
							return
						}
						n, err := client.Write(data.Data)
						if err != nil {
							if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
								continue
							}
							ctx.Log.Errorln(err)
							return
						}
						if n != len(data.Data) {
							ctx.Log.Println("n, len(data.Data): ", n, len(data.Data))
						}
					}
				}
			}()
		}(client)
	}
}

// safrp TCP服务端插件
func SafrpTCPServer(ctx *common.Context) {
	for {
		ctx.Expand.(Context).PipeConnControllor <- 1		// 控制safrp客户端与safrp服务端最大连接数
		id, _ := ctx.Expand.(Context).ConnNumberPool.Get()
		client, err := ctx.Conn.(net.Listener).Accept()
		if err != nil {
			ctx.Log.Errorln(err)
		}
		ctx.Expand.(Context).ConnManage[id] = client
		go func(id int) {
			defer func() {
				ctx.Expand.(Context).ConnNumberPool.Put(id)
				ctx.Expand.(Context).ConnManage[id] = nil
				<-ctx.Expand.(Context).PipeConnControllor
			}()
			defer func() {
				for p := recover();p != nil;p = recover() {
					ctx.Log.Println(p)
				}
			}()
			c := ctx.Expand.(Context).ConnManage[id]
			if !checkConnectPassword(c) { // 如果配置密码不匹配，结束连接
				ctx.Log.Errorln("连接密码错误。。。")
				return
			}
			<- ctx.Expand.(Context).ConnClose[id]
		}(int(id))
	}
}

// TCP写数据插件
func SafrpTCPSend(ctx *common.Context) {
	for {
		select {
		case data := <- ctx.ReadDate:
			// 向某个管道写数据
			ctx.Expand.(Context).ConnManage[data.Number % int(ctx.Conf.(Config).PipeNum)].Write(data.Data)
		}
	}
}

// TCP读数据插件
func SafrpTCPRead(ctx *common.Context) {
	for {
		select {
		case data := <- ctx.ReadDate:
			fmt.Println(data)
		}
	}
}

// 通过密码登录插件
// 连接安全验证插件
func checkConnectPassword(c net.Conn) bool {
	return false
}
// 发送心跳包插件
func SendHeartbeat(ctx *common.Context) {
	n, err := ctx.Conn.(net.Conn).Write(TCPDataEnd)
	if n != len(TCPDataEnd) || err != nil {
		// 如果是断开了
		ctx.Log.Errorln(err)
	}
}
// 判断心跳包插件
func ReadHeartbeat() {

}
// 与safrp客户端交互的数据解析插件
func parsePackage(c net.Conn) {
	go func() {common.Run(func() {


	})}()
}
/*-------------- 功能性插件 -----------------*/
// 限流插件
// IP记录插件
// 将记录写入mq中
// 将记录写到数据库
// 黑名单插件
// 白名单插件
// 插件接口
func plugInInterface(ctx *common.Context) {

}

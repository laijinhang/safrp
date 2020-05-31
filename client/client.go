package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"io"
	"log"
	"net"
	"safrp/common"
	"sync"
	"time"
)

type Config struct {
	ServerIP   string
	ServerPort string
	Password   string
	HTTPIP     string
	HTTPPort   string
	Protocol   string
	PipeNum    int
}

type Context struct {
	FromSafrpServer []chan common.DataPackage
	ToSafrpServer   []chan common.DataPackage
	FromProxyServer chan common.DataPackage
	ToProxyServer   chan common.DataPackage
}

var conf Config
var BufSize = 1024 * 1024 * 8
var TCPDataEnd = []byte{'<', 'e', '>'}

type TCPData struct {
	ConnId int
	Data   []byte
}

var BufPool = sync.Pool{New: func() interface{} {
	return make([]byte, 1024*1024*8)
}}

func init() {
	cfg, err := ini.Load("./safrp.ini")
	if err != nil {
		log.Fatal("Fail to read file: ", err)
	}
	temp, _ := cfg.Section("server").GetKey("ip")
	conf.ServerIP = temp.String()
	temp, _ = cfg.Section("server").GetKey("port")
	conf.ServerPort = temp.String()
	temp, _ = cfg.Section("server").GetKey("password")
	conf.Password = temp.String()
	temp, _ = cfg.Section("proxy").GetKey("ip")
	conf.HTTPIP = temp.String()
	temp, _ = cfg.Section("proxy").GetKey("port")
	conf.HTTPPort = temp.String()
	temp, _ = cfg.Section("proxy").GetKey("protocol")
	conf.Protocol = temp.String()
	temp, _ = cfg.Section("").GetKey("pipe_num")
	conf.PipeNum = func(v int, e error) int {
		if e != nil {
			panic(e)
		}
		return v
	}(temp.Int())

	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableSorting:         false,
		SortingFunc:            nil,
		DisableLevelTruncation: true,
		QuoteEmptyFields:       false,
		FieldMap:               nil,
		CallerPrettyfier:       nil,
	})
	logrus.SetReportCaller(true)

	logrus.Infoln("load safrp.ini ...")
	logrus.Infoln("server-ip:", conf.ServerIP)
	logrus.Infoln("server-port:", conf.ServerPort)
	logrus.Infoln("server-password:", conf.Password)
	logrus.Infoln("http-ip:", conf.HTTPIP)
	logrus.Infoln("http-port:", conf.HTTPPort)
	logrus.Infoln()

	logrus.SetLevel(logrus.PanicLevel)
}

func main() {
	common.Run(func() {
		log := logrus.New()
		log.SetLevel(logrus.TraceLevel)
		log.SetFormatter(&logrus.TextFormatter{
			ForceColors:            true,
			FullTimestamp:          true,
			TimestampFormat:        "2006-01-02 15:04:05",
			DisableSorting:         false,
			SortingFunc:            nil,
			DisableLevelTruncation: true,
			QuoteEmptyFields:       false,
			FieldMap:               nil,
			CallerPrettyfier:       nil,
		})
		log.SetReportCaller(true)

		ctx := common.Context{
			Conf:       conf,
			Conn:       make([]net.Conn, conf.PipeNum+1),
			NumberPool: common.NewNumberPool(uint64(conf.PipeNum), uint64(1)),
			IP:         conf.ServerIP,
			Port:       conf.ServerPort,
			Log:        log,
			Protocol:   common.GetBaseProtocol(conf.Protocol),
			Expand:Context{
                FromSafrpServer: make([]chan common.DataPackage, conf.PipeNum),
                ToSafrpServer:   make([]chan common.DataPackage, conf.PipeNum),
                FromProxyServer: make(chan common.DataPackage, 3000),
                ToProxyServer:   make(chan common.DataPackage, 3000),
            },
		}

		common.Run(func() {
			// 对safrp客户端
			SafrpClient(&ctx)
		})
		//common.Run(func() {
		//    // 代理服务
		//    SafrpClient(&ctx)
		//})
	})
}

// 单例模式
var Server = common.NewSingle()
var safrpClient = common.NewSingle()

func SafrpClient(ctx *common.Context) {
	connManage := make(chan int, ctx.Conf.(Config).PipeNum)
	for {
		connManage <- 1 // 没有达到最大隧道数
		// 创建一个连接
		conn, err := net.Dial(ctx.Protocol, ctx.IP+":"+ctx.Port)
		if err != nil {
			ctx.Log.Errorln(err)
			time.Sleep(3 * time.Second)
			<-connManage
			continue
		}
		id, _ := ctx.NumberPool.Get()
		ctx.Conn.([]net.Conn)[id] = conn
		// 管道取数据
		go func(id uint64) {
			defer func() {
				for p := recover(); p != nil; p = recover() {
					ctx.Log.Println(p)
				}
				ctx.Conn.([]net.Conn)[id].Close() // 关闭连接
				ctx.NumberPool.Put(int(id))
				<-connManage // 当前管道减一
			}()
			ctx.Conn.([]net.Conn)[id].Write([]byte(ctx.Conf.(Config).Password)) // 发送密码
			buf := make([]byte, 1)
			ctx.Conn.([]net.Conn)[id].Read(buf) // 读取连接结果
			if buf[0] == '0' {
				ctx.Log.Println("密码错误。。。")
				return
			}
			ctx.Log.Println(fmt.Sprintf("编号：%d，连接成功。。。\n", id))

			connClose := make(chan bool)
			FromStream := make(chan []byte, 10)
			// 数据转发中心
			go common.DataProcessingCenter(FromStream,
				ctx.Expand.(Context).ToProxyServer,
				[]byte("<<end>>"),
				connClose)
			go func() {
				for {
					select {
					case pack := <-ctx.Expand.(Context).ToProxyServer:
						ctx.Log.Infoln(fmt.Sprintf("编号：%d, Data：%s\n", pack.Number, string(pack.Data)))
					}
				}
			}()
			// 取数据
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
						err := ctx.Conn.([]net.Conn)[id].SetReadDeadline(time.Now().Add(1 * time.Second))
						if err != nil {
							return
						}
						n, err := ctx.Conn.([]net.Conn)[id].Read(buf)
						if err != nil {
							if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
								continue
							}
							return
						}
						fmt.Println(string(buf[:n]))
						FromStream <- buf[:n]
					}
				}
			}()

			sendHeartbeat := make(chan bool, 1)
			nextSendDataTime := time.Now().Unix()
			for {
				select {
				case <-sendHeartbeat:
					ctx.Conn.([]net.Conn)[id].Write([]byte("<<end>>"))
					nextSendDataTime = time.Now().Unix()
				//case : // 发送数据

				default:
					if time.Now().Unix()-nextSendDataTime >= 60 {
						sendHeartbeat <- true
					}
				}
			}
			// 数据解析
			select {}
		}(id)
	}
	select {}
}

func ProxyClient(ctx *common.Context) {
}

// 通过密码登录插件
func sendConnectPassword(ctx *common.Context) {
	for i := 0; i < len(ctx.Conn.([]net.Conn)); i++ {
		ctx.Conn.([]net.Conn)[i].Read([]byte(ctx.Conf.(Config).Password)) // 发送连接密码
	}
}

// 统一处理心跳包

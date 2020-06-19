package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"io"
	"log"
	"net"
	"safrp/common"
	"strings"
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

	ToClient []chan common.DataPackage
	ReadConn []*func()
	Conn []*net.Conn
}

var conf Config
var BufSize = 1024 * 8
var TCPDataEnd = []byte{'<', 'e', '>'}

var BufPool = sync.Pool{New: func() interface{} {
	return make([]byte, 1024*8)
}}

/**
 * 加载safrp服务端配置文件
 * @param		nil
 * @return		nil
 * func loadConfig();
 */
func loadConfig() {
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
		DisableLevelTruncation: true,
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
	loadConfig()
	common.Run(func() {
		log := logrus.New()
		log.SetLevel(logrus.TraceLevel)
		log.SetFormatter(&logrus.TextFormatter{
			ForceColors:            true,
			FullTimestamp:          true,
			TimestampFormat:        "2006-01-02 15:04:05",
			DisableLevelTruncation: true,
		})
		log.SetReportCaller(true)

		ctx := common.Context{
			Conf:       conf,
			Conn:       make([]net.Conn, conf.PipeNum+1),
			NumberPool: common.NewNumberPool(uint64(conf.PipeNum), uint64(1)),
			IP:         conf.ServerIP,
			Port:       conf.ServerPort,
			Log:        log,
			Protocol:   common.GetL3Protocol(conf.Protocol),
			Expand:Context{
                FromSafrpServer: make([]chan common.DataPackage, conf.PipeNum+1),
                ToSafrpServer:   make([]chan common.DataPackage, conf.PipeNum+1),
                FromProxyServer: make(chan common.DataPackage, 3000),
                ToProxyServer:   make(chan common.DataPackage, 3000),
                ToClient: make([]chan common.DataPackage, conf.PipeNum+1),
                Conn: make([]*net.Conn, conf.PipeNum+1),
                ReadConn: make([]*func(), 3000),
            },
		}
		for i := 0;i <= ctx.Conf.(Config).PipeNum;i++ {
			ctx.Expand.(Context).FromSafrpServer[i] = make(chan common.DataPackage, 10)
			ctx.Expand.(Context).ToClient[i] = make(chan common.DataPackage, 10)
			ctx.Expand.(Context).ToSafrpServer[i] = make(chan common.DataPackage, 10)
		}

		go common.Run(func() {
			// 对safrp客户端
			SafrpClient(&ctx)
		})
		common.Run(func() {
		  // 代理服务
			ProxyClient(&ctx)
		})
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
						//ctx.Log.Infoln(fmt.Sprintf("编号：%d, Data：%s\n", pack.Number, string(pack.Data)))
						ctx.Expand.(Context).ToClient[pack.Number%ctx.Conf.(Config).PipeNum] <- pack
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
							if neterr, ok := err.(net.Error); (ok && neterr.Timeout()) || err == io.EOF {
								continue
							}
							return
						}
						FromStream <- buf[:n]
					}
				}
			}()
			//
			sendHeartbeat := make(chan bool, 1)
			nextSendDataTime := time.Now().Unix()
			for {
				select {
				case <-sendHeartbeat:
					err := ctx.Conn.([]net.Conn)[id].SetWriteDeadline(time.Now().Add(time.Second))
					if err != nil {
						return
					}
					_, err = ctx.Conn.([]net.Conn)[id].Write([]byte("<<end>>"))
					if err != nil {
						if neterr, ok := err.(net.Error); (ok && neterr.Timeout()) || err == io.EOF {
							continue
						}
						return
					}
					nextSendDataTime = time.Now().Unix()
				case pack := <-ctx.Expand.(Context).ToSafrpServer[id]: // 发送数据
					err := ctx.Conn.([]net.Conn)[id].SetWriteDeadline(time.Now().Add(time.Second))
					if err != nil {
						return
					}
					_, err = ctx.Conn.([]net.Conn)[id].Write([]byte(fmt.Sprintf("%d %s %s\r\n%s%s",  pack.Number, "1", pack.Status, string(pack.Data), "<<end>>")))
					if err != nil {
						if neterr, ok := err.(net.Error); (ok && neterr.Timeout()) || err == io.EOF {
							continue
						}
						return
					}
				default:
					if time.Now().Unix()-nextSendDataTime >= 60 {
						sendHeartbeat <- true
					}
					time.Sleep(time.Second)
				}
			}
			// 数据解析
			select {}
		}(id)
	}
	select {}
}

func ProxyClient(ctx *common.Context) {
	wg := sync.WaitGroup{}
	for i := 0;i < ctx.Conf.(Config).PipeNum;i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				for p := recover(); p != nil; p = recover() {
					logrus.Println("panic:", p)
				}
			}()

			for {
				select {
				case data := <-ctx.Expand.(Context).ToClient[id]:
					if strings.Index(data.Status, "close") != -1 {
						ctx.Expand.(Context).ReadConn[data.Number] = nil
						ctx.Log.Infoln("编号：", data.Number, "连接关闭")
						ctx.Expand.(Context).Conn[id] = nil
						continue
					}
					if ctx.Expand.(Context).Conn[id] == nil {
						conn, err := net.Dial("tcp", ctx.Conf.(Config).HTTPIP+":"+ctx.Conf.(Config).HTTPPort)
						if err != nil {
							ctx.Log.Errorln(err)
							break
						}
						ctx.Expand.(Context).Conn[id] = &conn
					}
					n, err := (*ctx.Expand.(Context).Conn[id]).Write(data.Data)
					if err != nil {
						ctx.Log.Errorln(err)
						break
					}
					if n != len(data.Data) {
						ctx.Log.Errorln(n, "!=", len(data.Data))
					}
					ctx.Log.Infoln(ctx.Expand.(Context).ReadConn[data.Number] == nil)
					if ctx.Expand.(Context).ReadConn[data.Number] == nil {
						// 为什么注释的这一段代码是会锁死？？？
						//*(ctx.Expand.(Context).ReadConn[data.Number]) = func() {
						//	ctx.Log.Printf("编号：%d启动读。。。\n", data.Number)
						//	defer ctx.Log.Printf("编号：%d结束读。。。\n", data.Number)
						//	for {
						//		if ctx.Expand.(Context).ReadConn[data.Number] == nil {
						//			return
						//		}
						//		buf := make([]byte, 10240)
						//		n, err := (*ctx.Expand.(Context).Conn[id]).Read(buf)
						//		if err != nil {
						//			ctx.Log.Errorln(err)
						//			break
						//		}
						//		ctx.Expand.(Context).ToSafrpServer[id] <- common.DataPackage{
						//			Number: data.Number,
						//			Status: "open",
						//			Data:   buf[:n],
						//		}
						//	}
						//}
						f1 := func() {
							ctx.Log.Printf("编号：%d启动读。。。\n", data.Number)
							defer ctx.Log.Printf("编号：%d结束读。。。\n", data.Number)
							for {
								if ctx.Expand.(Context).ReadConn[data.Number] == nil {
									return
								}
								buf := make([]byte, 10240)
								n, err := (*ctx.Expand.(Context).Conn[id]).Read(buf)
								if err != nil {
									ctx.Log.Errorln(err)
									break
								}
								ctx.Expand.(Context).ToSafrpServer[id] <- common.DataPackage{
									Number: data.Number,
									Status: "open",
									Data:   buf[:n],
								}
							}
						}
						ctx.Expand.(Context).ReadConn[data.Number] = &f1
						ctx.Log.Infoln((*(ctx.Expand.(Context).ReadConn[data.Number])))
						go (*(ctx.Expand.(Context).ReadConn[data.Number]))()
					}
				}
			}
		}(i)
	}
	wg.Wait()
}

// 通过密码登录插件
func sendConnectPassword(ctx *common.Context) {
	for i := 0; i < len(ctx.Conn.([]net.Conn)); i++ {
		ctx.Conn.([]net.Conn)[i].Read([]byte(ctx.Conf.(Config).Password)) // 发送连接密码
	}
}
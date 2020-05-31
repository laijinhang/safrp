package main

import (
	"context"
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
	Password     string
	PipeNum      int
}

type Context struct {
	ConnManage         []net.Conn // 管理与safrp之间的连接
	ConnClose          []chan bool
	ConnNumberPool     *common.NumberPool // 连接编号池
	ConnDataChan       []chan common.DataPackage
	SafrpSendChan      []chan common.DataPackage // 连接
	PipeConnControllor chan int                  // 限制safrp客户端与safrp服务端之间的连接数
}

var confs []Config
var BufSize = 1024 * 10 * 8
var BufPool = sync.Pool{New: func() interface{} { return make([]byte, BufSize) }}

const (
	TCPStreamLength   = 1024 * 8 // 1个TCP窗口大小
	UDPDatagramLength = 1472     // 1个UDP报文长度

	DataEnd = "<<end>>"
)

var TCPDataEnd = []byte{'<', 'e', '>'}

func init() {
	cfg, err := ini.Load("./safrp.ini")
	if err != nil {
		logrus.Fatal("Fail to read file: ", err)
	}

	for i := 1; len(cfg.Section(fmt.Sprintf("pipe%d", i)).Keys()) != 0; i++ {
		pipeConf := cfg.Section(fmt.Sprintf("pipe%d", i))
		confs = append(confs, Config{
			IP:           pipeConf.Key("ip").String(),
			ExtranetPort: pipeConf.Key("extranet_port").String(),
			ServerPort:   pipeConf.Key("server_port").String(),
			Protocol:     pipeConf.Key("protocol").String(),
			Password:     pipeConf.Key("password").String(),
			PipeNum: func(v int, e error) int {
				if e != nil {
					panic(e)
				}
				return v
			}(pipeConf.Key("pipe_num").Int()),
		})
	}
}

func main() {
	common.Run(func() {
		// 启动 pipe组件
		for i := 0; i < len(confs); i++ {
			go func(i int) {
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
				log.Printf("启动pipe%d\n", i+1)

				ctx := common.Context{
					Conf:       confs[i],
					UnitId:     i + 1,
					NumberPool: common.NewNumberPool(3000, 1),
					SendData:   make(chan common.DataPackage, 1000),
					ReadDate:   make(chan common.DataPackage, 1000),
					Log:        log,
					Protocol:   common.GetBaseProtocol(confs[i].Protocol),
				}
				ctx1 := ctx
				ctx2 := ctx

				sendChan := make([]chan common.DataPackage, ctx.Conf.(Config).PipeNum)
				for i := 0;i < ctx.Conf.(Config).PipeNum;i++ {
					sendChan[i] = make(chan common.DataPackage)
				}
				connChan := make([]chan common.DataPackage, 3001)

				ctx1.Expand = Context{
					ConnManage:         make([]net.Conn, ctx.Conf.(Config).PipeNum+1),
					PipeConnControllor: make(chan int, ctx.Conf.(Config).PipeNum),
					SafrpSendChan:      sendChan,
					ConnNumberPool:     common.NewNumberPool(uint64(ctx.Conf.(Config).PipeNum), uint64(1)),
					ConnDataChan:       connChan}
				ctx2.Expand = Context{
					ConnManage:         make([]net.Conn, ctx.Conf.(Config).PipeNum+1),
					ConnClose:          make([]chan bool, ctx.Conf.(Config).PipeNum+1),
					SafrpSendChan:		sendChan,
					PipeConnControllor: make(chan int, ctx.Conf.(Config).PipeNum),
					ConnNumberPool:     common.NewNumberPool(uint64(ctx.Conf.(Config).PipeNum), uint64(1)),
					ConnDataChan:       connChan}

				es := common.UnitFactory(ctx.Conf.(Config).Protocol, ctx.Conf.(Config).IP, ctx.Conf.(Config).ExtranetPort)
				ss := common.UnitFactory(ctx.Conf.(Config).Protocol, ctx.Conf.(Config).IP, ctx.Conf.(Config).ServerPort)

				extranetServer.Register(&ctx1, &es)
				safrpServer.Register(&ctx2, &ss)

				go common.Run(func() {
					// 对外
					ExtranetServer(&ctx1)
				})
				//go func() {	// 测试
				//	var number int
				//	var buf string
				//	for {
				//		fmt.Print("输入编号：")
				//		fmt.Scan(&number)
				//		fmt.Print("输入数据：")
				//		fmt.Scan(&buf)
				//		fmt.Println("发送", number % ctx.Conf.(Config).PipeNum)
				//
				//		ctx2.Expand.(Context).SafrpSendChan[number % ctx.Conf.(Config).PipeNum] <- common.DataPackage{
				//			Number: number,
				//			Data:   []byte(buf)}
				//		fmt.Println("阻塞结束")
				//	}
				//}()
				go common.Run(func() {
					ctxExit, chanExit := context.WithCancel(context.Background())
					defer func() {
						chanExit()	// 通知该服务下的所有协程退出
					}()
					ctx2.Ctx = ctxExit
					// 对safrp客户端
					SafrpServer(&ctx2)
				})
			}(i)
		}
		select {}
	})
}

func ExtranetServer(ctx *common.Context) {
	extranetServer.Get(ctx).Server(ctx, []func(ctx *common.Context){
		common.TCPListen,
		ExtranetTCPServer,
	})
}

func SafrpServer(ctx *common.Context) {
	safrpServer.Get(ctx).Server(ctx, []func(ctx *common.Context){
		common.TCPListen,
	})
	safrpServer.Get(ctx).Server(ctx, []func(ctx *common.Context){
		SafrpTCPServer,
	})
}

// 单例模式
var extranetServer = common.NewSingle()
var safrpServer = common.NewSingle()

/*--------- 核心插件 -----------*/

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
			for c := 0; num == -1; c++ {
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
						err := client.SetReadDeadline(time.Now().Add(1 * time.Second))
						if err != nil {
							return
						}
						n, err := client.Read(buf)
						if err != nil {
							if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
								continue
							}
							return
						}
						// 向管道分发请求
						ctx.Expand.(Context).SafrpSendChan[num % ctx.Conf.(Config).PipeNum] <- common.DataPackage{
							Number: num,
							Data:   buf[:n]}
					}
				}
			}()
			// 写
			func() {
				defer func() {
					connClose <- true
				}()
				fmt.Println(len(ctx.Expand.(Context).ConnDataChan))
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
							ctx.Log.Errorln(err)
							if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
								continue
							}
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
		client, err := ctx.Conn.(net.Listener).Accept()
		if err != nil {
			ctx.Log.Errorln(err)
			continue
		}
		select {
		case ctx.Expand.(Context).PipeConnControllor <- 1:	// 控制safrp客户端与safrp服务端最大连接数
			id, _ := ctx.Expand.(Context).ConnNumberPool.Get()
			ctx.Log.Infoln(fmt.Sprintf("管道：%d 连接。。。", id))
			ctx.Expand.(Context).ConnManage[id] = client
			go func(id int) {
				defer func() {
					//ctx.Expand.(Context).ConnManage[id].Close()	// 关闭对应连接
					ctx.Expand.(Context).ConnManage[id % ctx.Conf.(Config).PipeNum] = nil   // 对应连接器清空
					ctx.Expand.(Context).ConnNumberPool.Put(id) // 编号放回到编号池
					<-ctx.Expand.(Context).PipeConnControllor
				}()
				defer func() {
					//for p := recover(); p != nil; p = recover() {
					//	ctx.Log.Println(p)
					//}
				}()
				c := ctx.Expand.(Context).ConnManage[id]
				ctx.Log.Infoln(fmt.Sprintf("管道：%d 验证密码。。。", id))
				if !checkConnectPassword(c, ctx) { // 如果配置密码不匹配，结束连接
					c.Write([]byte{'0'})
					ctx.Log.Errorln("连接密码错误。。。")
					return
				}
				c.Write([]byte{'1'})
				ctx.Log.Infoln(fmt.Sprintf("管道：%d 建立成功。。。", id))

				ExitChan := make(chan bool)
				FromStream := make(chan []byte, 100)
				ToStream := ctx.Expand.(Context).ConnDataChan[id]
				go common.DataProcessingCenter(FromStream, ToStream, []byte(DataEnd), ExitChan)
				// 从safrp客户端读数据
				go func() {
					defer func() {
						for err := recover(); err != nil; err = recover() {
							fmt.Println(err)
						}
						ExitChan <- true
					}()
					defer c.Close()

					var err error
					buf := make([]byte, 1024)
					for {
						err = c.SetReadDeadline(time.Now().Add(60 * time.Second))
						if err != nil {
							ctx.Expand.(Context).ConnClose[id] <- true
							ctx.Log.Errorln(err)
							return
						}
						n, err := c.Read(buf)
						if err != nil {
							ctx.Log.Errorln(err)
							if neterr, ok := err.(net.Error);(ok && neterr.Timeout()) || err == io.EOF {
								ctx.Log.Errorln(neterr, err)
								continue
							}
							return
						}
						FromStream <- buf[:n] // 发往数据处理中心
					}
				}()
				// 向safrp客户端写数据
				for {
					fmt.Println("启动")
					select {
					case <-ctx.Expand.(Context).ConnClose[id % ctx.Conf.(Config).PipeNum]:
						fmt.Println("退出")
						return
					case data := <-ctx.Expand.(Context).SafrpSendChan[id % ctx.Conf.(Config).PipeNum]:
						fmt.Println(data)
						//_, err = c.Write([]byte(fmt.Sprintf("%d %s\n%s%s", data.Number, "ip", string(data.Data), DataEnd)))
						_, err = c.Write([]byte(data.Data))
						if err != nil {
							ctx.Log.Errorln(err)
							if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
								continue
							}
							return
						}
					}
				}

			}(int(id))
		default:
			client.Close()
		}
	}
}

// 连接安全验证插件
func checkConnectPassword(c net.Conn, ctx *common.Context) bool {
	buf := make([]byte, 1024)
	n, _ := c.Read(buf)
	if string(buf[:n]) == ctx.Conf.(Config).Password {
		return true
	}
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
	go func() {
		common.Run(func() {

		})
	}()
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

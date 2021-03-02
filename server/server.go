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

var confs []common.Config
var BufSize = 1024 * 100 * 8
var BufPool = sync.Pool{New: func() interface{} { return make([]byte, BufSize) }}

const (
	TCPStreamLength   = 1024 * 8 // 1个TCP窗口大小
	UDPDatagramLength = 1472     // 1个UDP报文长度

	DataEnd = "<<end>>"
)

var TCPDataEnd = []byte{'<', 'e', '>'}

/**
 * 加载safrp服务端配置文件
 * @param		nil
 * @return		nil
 * func loadConfig();
 */
func loadConfig() {
	cfg, err := ini.Load("./safrp.ini")
	if err != nil {
		logrus.Fatal("Fail to read file: ", err)
	}

	for i := 1; len(cfg.Section(fmt.Sprintf("pipe%d", i)).Keys()) != 0; i++ {
		pipeConf := cfg.Section(fmt.Sprintf("pipe%d", i))
		confs = append(confs, common.Config{
			IP:           pipeConf.Key("ip").String(),
			ExtranetPort: pipeConf.Key("extranet_port").String(),
			ExtranetConnNum: func(v int, e error) int {
				if e != nil {
					panic(e)
				}
				return v
			}(pipeConf.Key("extranet_conn_num").Int()),
			ServerPort: pipeConf.Key("server_port").String(),
			Protocol:   pipeConf.Key("protocol").String(),
			Password:   pipeConf.Key("password").String(),
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
	loadConfig()
	common.Run(func() {
		// 启动 pipe组件
		for i := 0; i < len(confs); i++ {
			go pipeUnit(i)
		}
		select {}
	})
}

func pipeUnit(number int) {
	log := logrus.New()
	log.SetLevel(logrus.TraceLevel)
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: true,
	})
	log.SetReportCaller(true)
	log.Printf("启动pipe%d\n", number+1)

	ctx := common.Context{
		Conf:       confs[number],
		UnitId:     number + 1,
		NumberPool: common.NewNumberPool(uint64(confs[number].ExtranetConnNum), 1),
		SendData:   make(chan common.DataPackage, 1000),
		ReadDate:   make(chan common.DataPackage, 1000),
		Log:        log,
		Protocol:   common.GetL3Protocol(confs[number].Protocol),
	}
	ctx1 := ctx
	ctx2 := ctx

	sendChan := make([]chan common.DataPackage, ctx.Conf.(common.ServerConfig).PipeNum)
	readChan := make([]chan common.DataPackage, ctx.Conf.(common.ServerConfig)\.PipeNum)
	for i := 0; i < ctx.Conf.(common.ServerConfig)\.PipeNum; i++ {
		sendChan[i] = make(chan common.DataPackage)
		readChan[i] = make(chan common.DataPackage)
	}
	connChan := make([]chan common.DataPackage, confs[number].ExtranetConnNum)

	ctx1.Expand = common.Expand{
		PipeName: logrus.Fields{
			"pipe": fmt.Sprint(number + 1),
		},
		ConnManage:         make([]net.Conn, ctx.Conf.(common.ServerConfig)\.PipeNum+1),
		PipeConnControllor: make(chan int, ctx.Conf.(common.Config).PipeNum),
		SafrpSendChan:      sendChan,
		SafrpReadChan:      readChan,
		ConnNumberPool:     common.NewNumberPool(uint64(ctx.Conf.(common.Config).PipeNum), uint64(1)),
		ConnDataChan:       connChan}
	ctx2.Expand = common.Expand{
		PipeName: logrus.Fields{
			"pipe": fmt.Sprint(number + 1),
		},
		ConnManage:         make([]net.Conn, ctx.Conf.(common.Config).PipeNum+1),
		ConnClose:          make([]chan bool, ctx.Conf.(common.Config).PipeNum+1),
		SafrpSendChan:      sendChan,
		SafrpReadChan:      readChan,
		PipeConnControllor: make(chan int, ctx.Conf.(common.Config).PipeNum),
		ConnNumberPool:     common.NewNumberPool(uint64(ctx.Conf.(common.Config).PipeNum), uint64(1)),
		ConnDataChan:       connChan}

	es := common.UnitFactory(ctx.Conf.(common.Config).Protocol, ctx.Conf.(common.Config).IP, ctx.Conf.(common.Config).ExtranetPort)
	ss := common.UnitFactory(ctx.Conf.(common.Config).Protocol, ctx.Conf.(common.Config).IP, ctx.Conf.(common.Config).ServerPort)

	extranetServer.Register(&ctx1, &es)
	safrpServer.Register(&ctx2, &ss)

	go func() {
		// 对外
		ExtranetServer(&ctx1)
	}()
	func() {
		ctxExit, chanExit := context.WithCancel(context.Background())
		defer func() {
			chanExit() // 通知该服务下的所有协程退出
		}()
		ctx2.Ctx = ctxExit
		// 对safrp客户端
		SafrpServer(&ctx2)
	}()
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

func ExtranetUDPServer(ctx *common.Context) {

}

func SafrpUDPServer(ctx *common.Context) {}

// 单例模式
var extranetServer = common.NewSingle()
var safrpServer = common.NewSingle()

/**
 * 加载服务（工厂模式）
 * @param		server string, typ int
 * @return		func(ctx *common.Context)
 * func ServerFactory(server string, typ int) *func(ctx *common.Context);
 */
func ServerFactory(server string, typ int) *func(ctx *common.Context) {
	legality := typ == 1 || typ == 2
	if !legality {
		return nil
	}
	var serFunc func(ctx *common.Context)
	if typ == 1 {
		switch server {
		case "udp":
			serFunc = ExtranetUDPServer
		case "tcp":
			serFunc = ExtranetTCPServer
		case "http":
			serFunc = ExtranetTCPServer
		}
	} else {
		switch server {
		case "udp":
			serFunc = SafrpUDPServer
		case "tcp":
			serFunc = SafrpTCPServer
		case "http":
			serFunc = SafrpTCPServer
		}
	}
	return &serFunc
}

/*-------------- 核心插件 -----------------*/
/**
 * safrp TCP对外服务插件
 * @param		nil
 * @return		nil
 * func loadConfig();
 */
func ExtranetTCPServer(ctx *common.Context) {
	for {
		client, err := ctx.Conn.(net.Listener).Accept()
		if err != nil {
			continue
		}
		go func(client net.Conn) {
			defer func() {
				for p := recover(); p != nil; p = recover() {
					ctx.Log.WithFields(ctx.Expand.PipeName).Println(p)
				}
			}()
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
			defer func(number int) {
				// 通知safrp客户端，该临时编号已被回收
				ctx.Log.WithFields(ctx.Expand.PipeName).Infoln("编号：", num, "连接关闭")

				ctx.Expand.SafrpSendChan[num%ctx.Conf.(common.Config).PipeNum] <- common.DataPackage{
					Number: num,
					Data:   []byte(fmt.Sprintf("%d %s close\r\n%s", num, client.RemoteAddr(), DataEnd))}
				ctx.NumberPool.Put(number)
				close(ctx.Expand.ConnDataChan[num])
				ctx.Expand.ConnDataChan[num] = nil
			}(num)
			ctx.Log.WithFields(ctx.Expand.PipeName).Println("编号：", num)

			ctx.Expand.ConnDataChan[num] = make(chan common.DataPackage, 10)
			connCloseRead, cancelRead := context.WithCancel(context.Background())
			connCloseWrite, cancelWrite := context.WithCancel(context.Background())

			wg := sync.WaitGroup{}
			wg.Add(2)
			// 读
			go func() {
				buf := BufPool.Get().([]byte)
				defer func() {
					BufPool.Put(buf)
					cancelWrite()
					wg.Done()
				}()
				for {
					select {
					case <-connCloseRead.Done():
						return
					default:
						// 一直读，直到连接关闭
						err := client.SetReadDeadline(time.Now().Add(1 * time.Second))
						if err != nil {
							return
						}
						n, err := client.Read(buf)
						if err != nil {
							if err == io.EOF {
								return
							}
							if neterr, ok := err.(net.Error); (ok && neterr.Timeout()) || err == io.EOF {
								continue
							}
							ctx.Log.WithFields(ctx.Expand.PipeName).Errorln(err)
							return
						}
						// 向管道分发请求
						ctx.Expand.SafrpSendChan[num%ctx.Conf.(common.Config).PipeNum] <- common.DataPackage{
							Number: num,
							Data:   []byte(fmt.Sprintf("%d %s open\r\n%s%s", num, client.RemoteAddr(), string(buf[:n]), DataEnd))}
					}
				}
			}()
			// 写
			go func() {
				defer func() {
					cancelRead()
					wg.Done()
				}()
				for {
					if ctx.Expand.ConnDataChan[num] == nil {
						return
					}
					select {
					case <-connCloseWrite.Done():
						return
					case data := <-ctx.Expand.ConnDataChan[num]:
						if len(data.Data) == 0 {
							continue
						}
						// 有数据来的话，就一直写，直到连接关闭
						err := client.SetWriteDeadline(time.Now().Add(time.Second))
						if err != nil {
							ctx.Log.WithFields(ctx.Expand.PipeName).Errorln(err)
							return
						}
						n, err := client.Write(data.Data)
						if err != nil {
							if err == io.EOF {
								return
							}
							if neterr, ok := err.(net.Error); (ok && neterr.Timeout()) || err == io.EOF {
								continue
							}
							ctx.Log.WithFields(ctx.Expand.PipeName).Errorln(err)
							return
						}
						if n != len(data.Data) {
							ctx.Log.WithFields(ctx.Expand.PipeName).Println("n, len(data.Data): ", n, len(data.Data))
						}
					}
				}
			}()
			wg.Wait()
		}(client)
	}
}

/**
 * safrp TCP服务端插件
 * @param		nil
 * @return		nil
 * func loadConfig();
 */
func SafrpTCPServer(ctx *common.Context) {
	for {
		client, err := ctx.Conn.(net.Listener).Accept()
		if err != nil {
			ctx.Log.WithFields(ctx.Expand.PipeName).Errorln(err)
			continue
		}
		select {
		case ctx.Expand.PipeConnControllor <- 1: // 控制safrp客户端与safrp服务端最大连接数
			id, _ := ctx.Expand.ConnNumberPool.Get()
			ctx.Log.WithFields(ctx.Expand.PipeName).Infoln(fmt.Sprintf("管道：%d 连接。。。", id))
			ctx.Expand.ConnManage[id] = client
			go func(id int) {
				defer func() {
					//ctx.Expand.(Context).ConnManage[id].Close()	// 关闭对应连接
					ctx.Expand.ConnManage[id%ctx.Conf.(common.Config).PipeNum] = nil // 对应连接器清空
					ctx.Expand.ConnNumberPool.Put(id)                         // 编号放回到编号池
					<-ctx.Expand.PipeConnControllor
				}()
				defer func() {
					//for p := recover(); p != nil; p = recover() {
					//	ctx.Log.Println(p)
					//}
				}()
				c := ctx.Expand.ConnManage[id]
				ctx.Log.WithFields(ctx.Expand.PipeName).Infoln(fmt.Sprintf("管道：%d 验证密码。。。", id))
				if !checkConnectPassword(c, ctx) { // 如果配置密码不匹配，结束连接
					c.Write([]byte{'0'})
					ctx.Log.WithFields(ctx.Expand.PipeName).Errorln("连接密码错误。。。")
					return
				}
				c.Write([]byte{'1'})
				ctx.Log.WithFields(ctx.Expand.PipeName).Infoln(fmt.Sprintf("管道：%d 建立成功。。。", id))

				ExitChan := make(chan bool)
				FromStream := make(chan []byte, 100)
				ToStream := ctx.Expand.SafrpReadChan[id%ctx.Conf.(common.Config).PipeNum]
				go func(num int) {
					for {
						select {
						case data := <-ToStream:
							if ctx.Expand.ConnDataChan[data.Number] != nil {
								ctx.Expand.ConnDataChan[data.Number] <- common.DataPackage{
									Number: data.Number,
									Data:   data.Data,
									Status: data.Status,
								}
							}
						}
					}
				}(id)
				go common.DataProcessingCenter(FromStream, ToStream, []byte(DataEnd), ExitChan)
				// 从safrp客户端读数据
				go func() {
					defer func() {
						for err := recover(); err != nil; err = recover() {
							fmt.Println(err)
						}
						ctx.Log.WithFields(ctx.Expand.PipeName).Infoln(fmt.Sprintf("管道：%d 连接关闭。。。", id))
						ExitChan <- true
					}()
					defer c.Close()

					var err error
					buf := make([]byte, 1<<23)
					for {
						err = c.SetReadDeadline(time.Now().Add(60 * time.Second))
						if err != nil {
							ctx.Expand.ConnClose[id] <- true
							ctx.Log.WithFields(ctx.Expand.PipeName).Errorln(err)
							return
						}
						n, err := c.Read(buf)
						if err != nil {
							if neterr, ok := err.(net.Error); (ok && neterr.Timeout()) || err == io.EOF {
								continue
							}
							ctx.Log.WithFields(ctx.Expand.PipeName).Errorln(err)
							return
						}
						FromStream <- buf[:n] // 发往数据处理中心
					}
				}()
				// 向safrp客户端写数据
				for {
					select {
					case <-ctx.Expand.ConnClose[id%ctx.Conf.(common.Config).PipeNum]:
						return
					case data := <-ctx.Expand.SafrpSendChan[id%ctx.Conf.(common.Config).PipeNum]:
						//_, err = c.Write([]byte(fmt.Sprintf("%d %s\n%s%s", data.Number, "ip", string(data.Data), DataEnd)))
						_, err = c.Write(data.Data)
						if err != nil {
							ctx.Log.WithFields(ctx.Expand.PipeName).Errorln(err)
							if neterr, ok := err.(net.Error); (ok && neterr.Timeout()) || err == io.EOF {
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

/**
 * 连接安全验证插件
 * @param		nil
 * @return		nil
 * func loadConfig();
 */
func checkConnectPassword(c net.Conn, ctx *common.Context) bool {
	buf := make([]byte, 1024)
	n, _ := c.Read(buf)
	if string(buf[:n]) == ctx.Conf.(common.ServerConfig).Password {
		return true
	}
	return false
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

package common

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"sync"
	"time"
)

/**
 * 初始化logrus配置
 * @param		nil
 * @return		nil
 * func initLogConfig();
 */
func initLogConfig() {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:               true,
		FullTimestamp:             true,
		TimestampFormat:           "2006-01-02 15:04:05",
		DisableLevelTruncation:    true,
	})
	logrus.SetReportCaller(true)
}

var onceExecFunc sync.Once
var restartTime = time.Duration(1)

/**
 * 支持panic重启的函数运行
 * @param		server func()	要执行的函数
 * @return		nil
 * func Run(server func());
 */
func Run(server func()) {
	onceExecFunc.Do(initLogConfig)
	resertRun := sync.WaitGroup{}
	for {
		resertRun.Add(1)
		go func() {
			defer resertRun.Done()
			defer func() {
				for p := recover(); p != nil; p = recover() {
					logrus.Println("panic:", p)
				}
			}()
			server()
		}()
		resertRun.Wait()
		time.Sleep(restartTime * time.Second)
	}
}


type single struct {
	lock sync.Mutex
	server map[*Context]Server
}

/**
 * 创建一个单利模式
 * @param		nil
 * @return		nil
 * func NewSingle() single;
 */
func NewSingle() single {
	return single{
		server: make(map[*Context]Server),
	}
}

/**
 * 注册一个服务
 * @param		ctx *Context, server *Server	上下文, 服务
 * @return		nil
 * func (s *single)Register(ctx *Context, server *Server);
 */
func (s *single)Register(ctx *Context, server *Server) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.server[ctx] = *server
}

/**
 * 取出一个服务
 * @param		ctx *Context	上下文
 * @return		nil
 * func (s *single)Get(ctx *Context) Server;
 */
func (s *single)Get(ctx *Context) Server {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.server[ctx]
}

/**
 * 组件工厂
 * @param		protocol, ip, port string	协议（TCP、TCP、UDP），ip，端口
 * @return		Server		生产一个服务
 * func UnitFactory(protocol, ip, port string) Server ;
 */
func UnitFactory(protocol, ip, port string) Server {
	switch protocol {
	case "tcp":
		return &TCPServer{IP:ip, Port:port}
	case "udp":
		return &UDPServer{IP:ip, Port:port}
	case "http":
		return &HTTPServer{IP: ip, Port:port}
	}
	return nil
}

/**
 * 获取协议对应的传输层协议
 * @param		protocol string		协议
 * @return		string				协议
 * func GetL3Protocol(protocol string) string;
 */
func GetL3Protocol(protocol string) string {
	switch strings.ToLower(protocol) {
	case "tcp", "http":
		return "tcp"
	case "udp":
		return "udp"
	}
	return ""
}

//
/**
 * 解析safrp客户端与safrp服务端之间的数据通信
 * @param		fromStream chan []byte, toStream chan DataPackage, dataEnd []byte, exitChan chan bool
 *				源数据流、目的数据流、一个数据的结束标识符，数据处理中心处理结束
 * @return		nil
 * func DataProcessingCenter(fromStream chan []byte, toStream chan DataPackage, dataEnd []byte, exitChan chan bool);
 */
func DataProcessingCenter(fromStream chan []byte, toStream chan DataPackage, dataEnd []byte, exitChan chan bool) {
	buf := []byte{}
	for {
		select {
		case <-exitChan:
			return
		case stream := <- fromStream:
			buf = append(buf, stream...)
			logrus.Infoln(string(buf))
			for i := bytes.Index(buf, dataEnd);i != -1;i = bytes.Index(buf, dataEnd) {
				tempBuf := bytes.SplitN(buf, dataEnd, 2)
				buf = nil
				if !bytes.HasSuffix(buf, dataEnd) {
					buf = tempBuf[1]
				}

				if len(tempBuf[0]) == 0 {
					continue
				}

				temp := bytes.Split(tempBuf[0], []byte{' '})
				tId, _ := strconv.Atoi(string(temp[0]))

				tB := []byte{}
				if len(bytes.SplitN(tempBuf[0], []byte("\r\n"), 2)) != 1 {
					tB = (bytes.SplitN(tempBuf[0], []byte("\r\n"), 2))[1]
				}
				status := "open"
				if len(temp) >= 3 && bytes.HasPrefix(temp[2], []byte("close")) {
					status = "close"
				}
				go func( id int, buf []byte, status string) {
					toStream <- DataPackage{
						Number: id,
						Data:   buf,
						Status: status,
					}
				}(tId, tB, status)
			}
		}
	}
}

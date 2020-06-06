package common

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"time"
)

func init() {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
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
	logrus.SetReportCaller(true)
}

func Run(server func()) {
	wg := sync.WaitGroup{}
	for {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				for p := recover(); p != nil; p = recover() {
					logrus.Println("panic:", p)
				}
			}()

			server()
		}()
		wg.Wait()
		time.Sleep(time.Second)
	}
}


type single struct {
	lock sync.Mutex
	server map[*Context]Server
}

func NewSingle() single {
	return single{
		server: make(map[*Context]Server),
	}
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

func GetBaseProtocol(protocol string) string {
	switch protocol {
	case "tcp", "http":
		return "tcp"
	case "udp":
		return "udp"
	}
	return ""
}

// 数据处理中心：safrp客户端发往safrp服务端
// 形参：源数据流、目的数据流、一个数据的结束标识符，数据处理中心处理结束
func DataProcessingCenter(FromStream chan []byte, ToStream chan DataPackage, DataEnd []byte, ExitChan chan bool) {
	buf := []byte{}
	for {
		select {
		case <-ExitChan:
			return
		case stream := <- FromStream:
			buf = append(buf, stream...)
			logrus.Infoln(string(buf))
			for i := bytes.Index(buf, DataEnd);i != -1;i = bytes.Index(buf, DataEnd) {
				tempBuf := bytes.SplitN(buf, DataEnd, 2)
				if bytes.HasSuffix(buf, DataEnd) {
					buf = nil
				} else {
					buf = tempBuf[1]
				}

				if len(tempBuf[0]) == 0 {
					continue
				}

				tId := 0
				temp := bytes.Split(tempBuf[0], []byte{' '})
				tId, _ = strconv.Atoi(string(temp[0]))

				tB := []byte{}
				if len(bytes.SplitN(tempBuf[0], []byte("\r\n"), 2)) == 1 {
					tB = []byte{}
				} else {
					tB = (bytes.SplitN(tempBuf[0], []byte("\r\n"), 2))[1]
				}
				status := "open"
				if len(temp) >= 3 {
					if bytes.HasPrefix(temp[2], []byte("close")) {
						status = "close"
					}
				}
				go func( id int, buf []byte, status string) {
					ToStream <- DataPackage{
						Number: id,
						Data:   buf,
						Status: status,
					}
				}(tId, tB, status)
			}
		}
	}
}

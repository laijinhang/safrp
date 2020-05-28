package common

import (
	"github.com/sirupsen/logrus"
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
package common

import (
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type Server interface {
	ReadServer(ctx *Context)
	SendServer(ctx *Context)
	Proxy() string
}

type TCPServer struct {
	IP string
	Port string
}

func (this *TCPServer) ReadServer(ctx *Context) {
	if c == nil {

	} else {
		listen, err := net.Listen("tcp", this.IP+":"+this.Port)
		logrus.Infoln("listen :" + this.IP + ":" + this.Port + " ...")
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
					for p := recover(); p != nil; p = recover() {
						logrus.Errorln(p)
					}
				}()
				defer c.Close()
				num := -1
				for c := 0; num == -1; c++ {
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
}

func (t *TCPServer) SendServer(ctx *Context) {}

func (t *TCPServer) Proxy() string {
	return "tcp"
}

type UDPServer struct {
	IP string
	Port string
}

func (u *UDPServer) ReadServer(ctx *Context) {}
func (u *UDPServer) SendServer(ctx *Context) {}
func (u *UDPServer) Proxy() string {
	return "udp"
}

type HTTPServer struct {
	IP string
	Port string
}

func (h *HTTPServer) ReadServer(ctx *Context) {}
func (h *HTTPServer) SendServer(ctx *Context) {}
func (h *HTTPServer) Proxy() string {
	return "http"
}


func ExtranetTCP() {

}

func SafrpServerTCP() {

}

func SafrpClientTCP() {

}

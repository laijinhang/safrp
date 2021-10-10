package common

import (
	"github.com/sirupsen/logrus"
	"net"
)

type Server interface {
	Run()
	GetConn() chan net.Conn
}

type TcpServer struct {
	ip string
	port string
	conns chan net.Conn
}

func NewTcpServer(ip, port string, connNum int) Server {
	return &TcpServer{
		ip:    ip,
		port:  port,
		conns: make(chan net.Conn, connNum),
	}
}

func (this *TcpServer) Run() {
	s, err := net.Listen("tcp", this.ip + ":" + this.port)
	if err != nil {
		panic(err)
	}
	for {
		c, err := s.Accept()
		if err != nil {
			logrus.Println(err)
			continue
		}
		this.conns <- c
	}
}

func (this *TcpServer) GetConn() chan net.Conn {
	if this.conns == nil {
		return nil
	}
	return this.conns
}

type Conn struct {
	SafrpClient net.Conn
	UserClient net.Conn
}

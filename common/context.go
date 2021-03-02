package common

import (
	"context"
	"github.com/sirupsen/logrus"
	common "http"
	"net"
)

type Context struct {
	Conf       interface{}
	UnitId     int
	NumberPool *NumberPool
	ReadDate   chan DataPackage
	SendData   chan DataPackage
	DateLength int
	IP         string
	Port       string
	Protocol   string
	Conn       interface{}
	Log        *logrus.Logger
	Expand     Expand
	Ctx        context.Context
}

type ServerConfig struct {
	IP              string
	ExtranetPort    string
	ExtranetConnNum int
	ServerPort      string
	Protocol        string
	Password        string
	PipeNum         int
}

type Expand struct {
	PipeName           logrus.Fields // pipe名
	ConnManage         []net.Conn    // 管理与safrp之间的连接
	ConnClose          []chan bool
	ConnNumberPool     *NumberPool // 连接编号池
	ConnDataChan       []chan DataPackage
	SafrpSendChan      []chan DataPackage // 连接
	SafrpReadChan      []chan DataPackage // 连接
	PipeConnControllor chan int           // 限制safrp客户端与safrp服务端之间的连接数
}

package main

import (
    "github.com/sirupsen/logrus"
    "gopkg.in/ini.v1"
    "log"
    "safrp/common"
    "sync"
    "time"
)

type Config struct {
    ServerIP string
    ServerPort string
    Password string
    HTTPIP string
    HTTPPort string
    Protocol string
    PipeNum int
}

var conf Config
var BufSize = 1024 * 1024 * 8
var TCPDataEnd = []byte{'<','e','>'}

type TCPData struct {
    ConnId int
    Data []byte
}

var BufPool = sync.Pool{New: func() interface{} {
    return make([]byte, 1024 * 1024 * 8)
}}

func init() {
    cfg, err := ini.Load("./safrp.ini")
    if err != nil {
        log.Fatal("Fail to read file: ", err)
    }
    temp, _ :=cfg.Section("server").GetKey("ip")
    conf.ServerIP = temp.String()
    temp, _ =cfg.Section("server").GetKey("port")
    conf.ServerPort = temp.String()
    temp, _ =cfg.Section("server").GetKey("password")
    conf.Password = temp.String()
    temp, _ =cfg.Section("proxy").GetKey("ip")
    conf.HTTPIP = temp.String()
    temp, _ =cfg.Section("proxy").GetKey("port")
    conf.HTTPPort = temp.String()
    temp, _ =cfg.Section("proxy").GetKey("protocol")
    conf.Protocol = temp.String()
    temp, _ =cfg.Section("").GetKey("pipe_num")
    conf.PipeNum = func(v int, e error) int {
        if e != nil {
            panic(e)
        }
        return v
    }(temp.Int())

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
        ctx := common.Context{
            Conf:       conf,
            UnitId:     0,
            NumberPool: nil,
            ReadDate:   nil,
            SendData:   nil,
            DateLength: 0,
            IP:         conf.ServerIP,
            Port:       conf.ServerPort,
            Protocol:   common.GetBaseProtocol(conf.Protocol),
        }

        common.Run(func() {
            // 对safrp服务端
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
    //connManage := make(chan int, ctx.Conf.(Config).PipeNum)
    for {
        common.TCPConnect(ctx)
        time.Sleep(time.Second)
    }
}

func ProxyClient(ctx *common.Context) {
}
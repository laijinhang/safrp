package main

import (
    "bytes"
    "github.com/sirupsen/logrus"
    "gopkg.in/ini.v1"
    "io"
    "log"
    "net"
    "safrp/common"
    "strconv"
    "sync"
    "time"
)

type Config struct {
    ServerIP string
    ServerPort string
    Password string
    HTTPIP string
    HTTPPort string
}

var conf Config
var BufSize = 1024 * 1024 * 8
var TCPDataEnd = []byte{'<','e','>'}

var tcpToServerStream = make(chan TCPData, 1000)
var tcpFromServerStream = make(chan TCPData, 1000)
var httpClient = [2001]common.HTTPClient{}

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
    temp, _ =cfg.Section("http").GetKey("ip")
    conf.HTTPIP = temp.String()
    temp, _ =cfg.Section("http").GetKey("port")
    conf.HTTPPort = temp.String()

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
        go common.Run(proxyClient)
        common.Run(Client)
    })
}

func proxyClient() {
    for {
        func() {
            defer func() {
                for p := recover();p != nil;p = recover() {
                    logrus.Println("panic:", p)
                }
                time.Sleep(3 * time.Second) // 防特殊情况下,一直重启,导致服务器内存资源消耗
            }()

            logrus.Infoln("safrp client ...")
            connNum := make(chan bool, 30)
            for i := 0; i < 30; i++ {
                connNum <- true
            }
            for {
                select {
                case <-connNum:
                    time.Sleep(3 * time.Second) // 缓慢启动和重启，防突然启动不可连接，导致CPU温度飙升
                    go func() {
                        defer func() {
                            connNum <- true
                        }()
                        defer func() {
                            for p := recover();p != nil;p = recover() {
                                logrus.Println("panic:", p)
                            }
                            time.Sleep(3 * time.Second) // 防特殊情况下,一直重启,导致服务器内存资源消耗
                        }()

                        conn, err := net.Dial("tcp", conf.ServerIP+":"+conf.ServerPort)
                        if err != nil {
                            logrus.Errorln(err)
                            time.Sleep(3 * time.Second)
                            return
                        }
                        defer conn.Close()
                        err = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
                        if err != nil {
                            logrus.Errorln(err)
                            return
                        }
                        _, err = conn.Write([]byte(conf.Password))
                        if err != nil {
                            logrus.Errorln(err)
                            return
                        }
                        err = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
                        if err != nil {
                            logrus.Errorln(err)
                            return
                        }
                        buf := make([]byte, 100)
                        _, err = conn.Read(buf)
                        if err != nil {
                            logrus.Errorln("密码错误...")
                            return
                        }
                        logrus.Infof("\n--------------------------\n%s\n---------------------\n", string(buf))

                        var wg sync.WaitGroup

                        var steamChan = make(chan []byte, 1000)
                        var dataChan = make(chan []byte, 1000)
                        var closeConn = make(chan bool, 5)

                        wg.Add(3)
                        go func() {
                            defer wg.Done()
                            common.Run(func() {
                                common.TCPSafrpStream(steamChan, dataChan, TCPDataEnd)
                            })
                        }()
                        go func() {
                            defer wg.Done()
                            common.Run(func() {
                                Read(conn, closeConn, steamChan, dataChan)
                            })
                        }()
                        go func() {
                            defer wg.Done()
                            common.Run(func() {
                                Send(conn, closeConn)
                            })
                        }()
                        wg.Wait()

                        logrus.Infoln("重连")
                    }()
                }
            }
        }()
    }
}

// 从 内网穿透服务器 读数据
func Read(c net.Conn, closeConn chan bool, streamChan, dataChan chan []byte) {
    defer func() {
        closeConn <- true
    }()

    buf := BufPool.Get().([]byte)
    defer func() {
        BufPool.Put(buf)
    }()
    go func() {
        defer func() {
            for p := recover();p != nil;p = recover() {
                logrus.Println("panic:", p)
            }
        }()
        for {
            select {
            case buf := <-dataChan:
                tBuf := bytes.SplitN(buf, []byte("\r\n"), 2)
                if len(tBuf) == 1 {
                    continue
                }
                tId := 0
                temp := bytes.Split(tBuf[0], []byte{' '})
                logrus.Infoln(string(tBuf[0]))
                logrus.Infoln(string(temp[1]))
                tId, _ = strconv.Atoi(string(temp[1]))
                //for i := 0;i < len(temp[1]) && temp[1][i] >= '0' && temp[1][i] <= '9';i++ {
                //    tId = tId * 10 + int(tBuf[1][i] - '0')
                //}
                logrus.Infoln("从safrp服务端", c.RemoteAddr(), "读取到数据,IP：" + string(temp[0]) + ",tId:", tId, ",data length:", len(tBuf[1]))

                if len(tBuf[1]) == 0 {
                    logrus.Infoln("编号：", tId, "断开。。。")
                    httpClient[tId].Addr = ""
                    httpClient[tId].Number = 0
                    httpClient[tId].Conn = nil
                    continue
                }
                if httpClient[tId].Addr != string(temp[0]) || (httpClient[tId].Addr == string(temp[0]) && httpClient[tId].Number != uint64(tId)) {
                    httpClient[tId].Addr = string(temp[0])
                    httpClient[tId].Number = uint64(64)
                    httpClient[tId].Dail(conf.HTTPIP + ":" + conf.HTTPPort)
                } else {
                    if httpClient[tId].Conn == nil {
                        httpClient[tId].Dail(conf.HTTPIP + ":" + conf.HTTPPort)
                    }
                }
                logrus.Infoln("临时编号:", tId)
                tcpFromServerStream <- TCPData{
                    ConnId: tId,
                    Data:   append([]byte{
                    }, tBuf[1]...),
                }
            }
        }
    }()
    for {
        err := c.SetReadDeadline(time.Now().Add(3 * time.Second))
        if err != nil {
            logrus.Errorln(err)
            return
        }
        n, err := c.Read(buf)
        if err != nil {
            logrus.Errorln(err)
            if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                 continue
            }
            return
        }

        streamChan <- buf[:n]
    }
}

// 往 内网穿透服务器 发数据
func Send(c net.Conn, closeConn chan bool) {
    HeartbeatB := make(chan []byte, 20)
    Heartbeat := make(chan bool, 1000)
    HeartbeatEnd := make(chan bool, 5)
    go func() { // 心跳包处理
       BeginTime := time.Now().Unix()
       for {
           select {
           case <-Heartbeat:
               BeginTime = time.Now().Unix()
           case <-HeartbeatEnd:  //退出
               return
           default:
               if time.Now().Unix() - BeginTime >= int64(3 * 60) { // 如果三分钟里，没有发送过数据，则发送心跳包
                   HeartbeatB <- TCPDataEnd
                   BeginTime = time.Now().Unix()
               }
               time.Sleep(1 * time.Second) // 使用休眠，CPU回到了12%左右，温度回归到正常58度上下
           }
           //runtime.Gosched()   // 放弃调度，不然CPU爆满，CPU温度飙升到80多度。使用放弃调度还是飙升到73度左右，CPU 81%的使用率
       }
    }()
    defer func() {
       logrus.Infoln("退出")
       HeartbeatEnd<-true
    }()
    for {
        select {
        case data := <- tcpToServerStream:  // 发送数据
            err := c.SetWriteDeadline(time.Now().Add(1 * time.Second))
            if err != nil {
                logrus.Errorln(err)
                return
            }
            logrus.Infoln("往safrp服务端", c.RemoteAddr(), "发送数据长度:", len(append([]byte(strconv.Itoa(data.ConnId) + "\r\n"), common.SafrpTCPPackage(data.Data, TCPDataEnd)...)))
            logrus.Infoln("临时id:",data.ConnId, len(data.Data), string(data.Data))
            _, err = c.Write(append([]byte(strconv.Itoa(data.ConnId) + "\r\n"), common.SafrpTCPPackage(data.Data, TCPDataEnd)...))
            if err != nil {
                logrus.Errorln(err)
                if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                    continue
                }
                return
            }
           Heartbeat <- true               // 心跳包时间重新计算
        case hb := <-HeartbeatB:            // 发送心跳包
           err := c.SetWriteDeadline(time.Now().Add(1 * time.Second))
           if err != nil {
               logrus.Errorln(err)
               return
           }
            logrus.Infoln("往safrp服务端", c.RemoteAddr(), "发送心跳包")
           _, err = c.Write(hb)
           if err != nil {
               logrus.Errorln(err)
               if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                   continue
               }
               return
           }
        case <- closeConn:                  // 连接断开时
           return
        }
    }
}

func Client() {
    for {
        func() {
            for p := recover();p != nil;p = recover() {
                logrus.Println("panic:", p)
            }
            select {
            case data := <-tcpFromServerStream:
                logrus.Infoln("临时编号:", data.ConnId)
                go func(d TCPData) {
                    defer func() {
                        for p := recover();p != nil;p = recover() {
                            logrus.Println("panic:", p)
                        }
                    }()
                    httpClient[d.ConnId].Write(d.Data)
                    buf, err := httpClient[d.ConnId].Read()
                    if err != nil {
                        logrus.Errorln(err)
                        return
                    }
                    tcpToServerStream <- TCPData{
                        ConnId: d.ConnId,
                        Data:   buf,
                    }
                }(data)
            }
        }()
    }
}

package main

import (
    "bytes"
    "fmt"
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
var BufSize = 1024 * 10 * 8
var TCPDataEnd = []byte("data_end;")

var tcpToServerStream = make(chan TCPData, 1000)
var tcpFromServerStream = make(chan TCPData, 1000)
var steamChan = make(chan []byte, 1000)
var dataChan = make(chan []byte, 1000)
var httpClient = [2001]common.HTTPClient{}

type TCPData struct {
    ConnId int
    Data []byte
}

var ConnPool = sync.Pool{}
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

    fmt.Println("load safrp.ini ...")
    fmt.Println("server-ip:", conf.ServerIP)
    fmt.Println("server-port:", conf.ServerPort)
    fmt.Println("server-password:", conf.Password)
    fmt.Println("http-ip:", conf.HTTPIP)
    fmt.Println("http-port:", conf.HTTPPort)
    fmt.Println()
}

func main() {
    wg := sync.WaitGroup{}
    go func() {
        defer func() {
            for p := recover();p != nil;p = recover() {
                fmt.Println(p)
            }
        }()
        common.TCPStream(steamChan, dataChan, TCPDataEnd)
    }()
    for {
        wg.Add(1)
        go func() {
            defer wg.Done()
            defer func() {
                for p := recover();p != nil;p = recover() {
                    fmt.Println(p)
                }
            }()
            go proxyClient()
            Client()
        }()
        wg.Wait()
    }
}

func proxyClient() {
    for {
        func() {
            defer func() {
                for p := recover();p != nil;p = recover() {
                    fmt.Println(p)
                }
            }()

            fmt.Println("safrp client ...")
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
                                fmt.Println(p)
                            }
                        }()

                        conn, err := net.Dial("tcp", conf.ServerIP+":"+conf.ServerPort)
                        if err != nil {
                            fmt.Println(err)
                            time.Sleep(3 * time.Second)
                            return
                        }
                        defer conn.Close()
                        err = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
                        if err != nil {
                            return
                        }
                        _, err = conn.Write([]byte(conf.Password))
                        if err != nil {
                            return
                        }
                        err = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
                        if err != nil {
                            return
                        }
                        buf := make([]byte, 100)
                        _, err = conn.Read(buf)
                        if err != nil {
                            fmt.Println("密码错误...")
                            return
                        }
                        fmt.Println(string(buf))

                        var closeConn = make(chan bool, 5)
                        go Read(conn, closeConn)
                        Send(conn, closeConn)
                        fmt.Println("重连")
                    }()
                }
            }
        }()
    }
}

// 从 内网穿透服务器 读数据
func Read(c net.Conn, closeConn chan bool) {
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
                fmt.Println(p)
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
                for i := 0;i < len(temp[1]);i++ {
                    if tBuf[0][i] != '\r' && tBuf[0][i] != '\n' {
                        tId = tId * 10 + int(tBuf[0][i] - '0')
                    }
                }
                fmt.Println("从safrp服务端", c.RemoteAddr(), "读取到数据,IP：" + string(temp[0]) + ",tId:", tId, ",data:", len(tBuf[1]))

                if len(tBuf[1]) == 0 {
                    fmt.Println("编号：", tId, "断开。。。")
                    httpClient[tId].Addr = ""
                    httpClient[tId].Conn = nil
                    continue
                }
                if httpClient[tId].Addr != string(temp[0]) {
                    httpClient[tId].Addr = string(temp[0])
                    httpClient[tId].Dail(conf.HTTPIP + ":" + conf.HTTPPort)
                }
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
            return
        }
        n, err := c.Read(buf)
        if err != nil {
            if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                 continue
            }
            return
        }

        steamChan <- buf[:n]
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
                   HeartbeatB <- append([]byte("1"),TCPDataEnd...)
                   BeginTime = time.Now().Unix()
               }
               time.Sleep(1 * time.Second) // 使用休眠，CPU回到了12%左右，温度回归到正常58度上下
           }
           //runtime.Gosched()   // 放弃调度，不然CPU爆满，CPU温度飙升到80多度。使用放弃调度还是飙升到73度左右，CPU 81%的使用率
       }
    }()
    defer func() {
       fmt.Println("退出")
       HeartbeatEnd<-true
    }()
    for {
        select {
        case data := <- tcpToServerStream:  // 发送数据
            err := c.SetWriteDeadline(time.Now().Add(1 * time.Second))
            if err != nil {
                return
            }
            fmt.Println("往safrp服务端", c.RemoteAddr(), "发送数据:", len(data.Data))
            _, err = c.Write(append([]byte(strconv.Itoa(data.ConnId) + "\r\n"), common.SafrpTCPPackage(data.Data, TCPDataEnd)...))
            if err != nil {
                if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                    continue
                }
                return
            }
           Heartbeat <- true               // 心跳包时间重新计算
        case hb := <-HeartbeatB:            // 发送心跳包
           err := c.SetWriteDeadline(time.Now().Add(1 * time.Second))
           if err != nil {
               return
           }
            fmt.Println("往safrp服务端", c.RemoteAddr(), "发送心跳包")
           _, err = c.Write(hb)
           if err != nil {
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
            defer func() {
                for p := recover();p != nil;p = recover() {
                    fmt.Println(p)
                }
            }()
            select {
            case data := <-tcpFromServerStream:
                go func(d TCPData) {
                    defer func() {
                        for err := recover(); err != nil; err = recover() {
                            fmt.Println(err)
                        }
                    }()
                    httpClient[d.ConnId].Write(d.Data)
                    buf, err := httpClient[d.ConnId].Read()
                    if err != nil {
                        fmt.Println(err)
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

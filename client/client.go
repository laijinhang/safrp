package main

import (
    "bytes"
    "fmt"
    "gopkg.in/ini.v1"
    "io"
    "log"
    "net"
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

var tcpToServerStream = make(chan TCPData, 1000)
var tcpFromServerStream = make(chan TCPData, 1000)

type TCPData struct {
    ConnId int
    Data []byte
}

var ConnPool = sync.Pool{}
var BufPool = sync.Pool{}

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
    for {
        wg.Add(1)
        go func() {
            defer wg.Done()
            defer func() {
                for v := recover();;v = recover() {
                    fmt.Println(v)
                }
            }()
            go proxyClient()
            go Client()
        }()
        wg.Wait()
    }
}

func proxyClient() {
    for {
        func() {
            defer func() {
                for err := recover(); err != nil; err = recover() {
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
                    go func() {
                        defer func() {
                            connNum <- true
                        }()
                        conn, err := net.Dial("tcp", conf.ServerIP+":"+conf.ServerPort)
                        if err != nil {
                            fmt.Println(err)
                            time.Sleep(3 * time.Second)
                            return
                        }
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

    tBuf := BufPool.Get()
    buf := []byte{}
    if tBuf == nil {
        buf = make([]byte, BufSize)
    } else {
        buf = tBuf.([]byte)
    }
    defer func() {
        BufPool.Put(buf)
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

        tBuf := bytes.SplitN(buf[:n], []byte("\r\n"), 2)
        tId := 0
        for i := 0;i < len(tBuf[0]);i++ {
            fmt.Println(tBuf[0])
            if tBuf[0][i] != '\r' && tBuf[0][i] != '\n' {
                tId = tId * 10 + int(tBuf[0][i] - '0')
            }
        }
        tcpFromServerStream <- TCPData{
            ConnId: tId,
            Data:   tBuf[1],
        }
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
                   HeartbeatB <- append([]byte("1"), []byte("data_end;")...)
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
            _, err = c.Write(append([]byte(strconv.Itoa(data.ConnId) + "\r\n"), append(data.Data, []byte("data_end;")...)...))
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
        select {
        case data := <- tcpFromServerStream:
            go func(d TCPData) {
                defer func() {
                    for err := recover();err != nil;err = recover(){
                    }
                }()
                c, err := net.Dial("tcp", conf.HTTPIP + ":" + conf.HTTPPort)
                fmt.Println(err)
                if err != nil {
                    return
                }
                go IntranetTransmitSend(c, d.Data)
                IntranetTransmitRead(c, d.ConnId)
            }(data)
        }
    }
}

// 从 内网服务器 读数据
func IntranetTransmitRead(c net.Conn, cId int) {
    buf := make([]byte, BufSize)
    for {
        err := c.SetReadDeadline(time.Now().Add(2 * time.Second))
        if err != nil {
            return
        }
        n, err := c.Read(buf)
        if n == 0 {
            if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                continue
            }
            return
        }
        //data := bytes.Replace(buf[:n], []byte(conf.ServerIP + ":" + conf.ServerPort), []byte("www.laijinhang.xyz"), 1)
        //fmt.Println(string(data))
        fmt.Println(string(buf[:n]))
        tcpToServerStream <- TCPData{
            ConnId: cId,
            Data:   buf[:n],
        }
        return
    }
    return
}

// 往 内网服务器 发数据
func IntranetTransmitSend(c net.Conn, data []byte) {
    fmt.Println("请求服务")
    //tBuf := bytes.SplitN(data, []byte("\r\n"), 3)
    //
    //tBuf[1] = []byte("Host: " + conf.HTTPIP)
    //if conf.HTTPPort != "80" {
    //   tBuf[1] = append(tBuf[1], []byte(":" + conf.HTTPPort)...)
    //}
    //data = bytes.Join(tBuf, []byte("\r\n"))
    //fmt.Println(string(data))
    //data = bytes.Replace(data, []byte(conf.ServerIP + ":10000"), []byte(conf.HTTPIP), 3)
    //fmt.Println(string(data))
    err := c.SetWriteDeadline(time.Now().Add(2 * time.Second))
    if err != nil {
        fmt.Println(err)
        return
    }
    _, err = c.Write(data)
    if err != nil {
        fmt.Println(err)
    }
}

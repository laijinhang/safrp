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
    HTTPIP string
    HTTPPort string
}

var conf Config
var BufSize = 1024 * 8

var tcpToServerStream = make(chan TCPData, 1000)
var tcpFromServerStream = make(chan TCPData, 1000)
var closeConn = make(chan bool, 2)

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
    temp, _ =cfg.Section("http").GetKey("ip")
    conf.HTTPIP = temp.String()
    temp, _ =cfg.Section("http").GetKey("port")
    conf.HTTPPort = temp.String()

    fmt.Println("load safrp.ini ...")
    fmt.Println("server-ip:", conf.ServerIP)
    fmt.Println("server-port:", conf.ServerPort)
    fmt.Println("http-ip:", conf.HTTPIP)
    fmt.Println("http-port:", conf.HTTPPort)
    fmt.Println()
}

func main() {
    go proxyClient()
    go Client()
    select {}
}

func proxyClient() {
    fmt.Println("safrp client ...")
    var lock sync.Mutex
    var num = 0
    for {
        lock.Lock()
        if num < 30 {
            num++
            lock.Unlock()
            go func() {
                defer func() {
                    lock.Lock()
                    num--
                    lock.Unlock()
                }()
                conn, err := net.Dial("tcp", conf.ServerIP + ":" + conf.ServerPort)
                if err != nil {
                    fmt.Println(err)
                    time.Sleep(3 * time.Second)
                    return
                }
                fmt.Println("connect success ...")
                go Read(conn)
                Send(conn)
                fmt.Println("重连")
            }()
        } else {
            lock.Unlock()
        }
    }
}

// 从 内网穿透服务器 读数据
func Read(c net.Conn) {
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
func Send(c net.Conn) {
    for {
        select {
        case data := <- tcpToServerStream:
            _, err := c.Write(append([]byte(strconv.Itoa(data.ConnId) + "\r\n"), data.Data...))
            if err != nil {
            }
        case <- closeConn:
            return
        }
    }
}

func Client() {
    for {
        select {
        case data := <- tcpFromServerStream:
            c, err := net.Dial("tcp", conf.HTTPIP + ":" + conf.HTTPPort)
            fmt.Println(err)
            if err != nil {
                return
            }
            go IntranetTransmitSend(c, data.Data)
            IntranetTransmitRead(c, data.ConnId)
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

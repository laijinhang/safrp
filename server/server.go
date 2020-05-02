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
}


var conf Config

var tcpToClientStream = make(chan TCPData, 1000)
var tcpFromClientStream [1001]interface{}

var ConnPool = sync.Pool{}
var BufPool = sync.Pool{}
var DeadTime = make([]chan interface{}, 1001)

func init() {
    for i := 1;i <= 1000;i++ {
        ConnPool.Put(i)
        tcpFromClientStream[i] = make(chan TCPData, 10)
    }
    cfg, err := ini.Load("./safrp.ini")
    if err != nil {
        log.Fatal("Fail to read file: ", err)
    }
    temp, _ :=cfg.Section("server").GetKey("ip")
    conf.ServerIP = temp.String()
    temp, _ =cfg.Section("server").GetKey("port")
    conf.ServerPort = temp.String()
}

type TCPData struct {
    ConnId int
    Data []byte
}

func main() {
    go proxyServer()    // 处理外网请求，短连接服务
    go server()         // 与穿透客户端进行交互，长连接服务
    select {}
}

// 处理 外网 请求
func proxyServer() {
    listen, err := net.Listen("tcp", ":10000")
    if err != nil {
        panic(err)
    }
    for {
        client, err := listen.Accept()
        if err != nil {
            fmt.Println(err)
            return
        }
        num := -1
        for num == -1 {
            tNum := ConnPool.Get()
            if tNum != nil {
                num = tNum.(int)
            } else {
                time.Sleep(50 * time.Millisecond)
            }
        }
        fmt.Println(client.RemoteAddr(), num)
        go func(c net.Conn, n int) {
            defer func() {
                ConnPool.Put(n)
                c.Close()
            }()
            go ExtranetRead(c, n)
            ExtranetSend(c, n)
        }(client, num)
    }
}

// 从 外网 接收数据
func ExtranetRead(c net.Conn, number int) {
    tBuf := BufPool.Get()
    buf := []byte{}
    if tBuf == nil {
        buf = make([]byte, 1024 * 10)
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
        tcpToClientStream <- TCPData{
            ConnId: number,
            Data:   buf[:n],
        }
    }
}

// 往 外网 响应数据
func ExtranetSend(c net.Conn, number int) {
    defer c.Close()

    BeginTime := time.Now().Unix()
    for {
        select {
        case data := <- tcpFromClientStream[number].(chan TCPData):
            err := c.SetWriteDeadline(time.Now().Add(3 * time.Second))
            if err != nil {
                continue
            }
            _, err = c.Write(data.Data)
            if err != nil {
                if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                    continue
                }
                return
            }
        default:
            if time.Now().Unix() - BeginTime >= int64(3) {
                c.Close()
                return
            }
        }
    }
}

// 处理穿透内网服务
func server() {
    listen, err := net.Listen("tcp", "127.0.0.1:8002")
    fmt.Println("safrp server listen :8002 ...")
    if err != nil {
        panic(err)
    }
    for {
        client, err := listen.Accept()
        if err != nil {
            fmt.Println(err)
            continue
        }
        fmt.Println("safrp client ", client.RemoteAddr(), "connect success ...")
        go func(c net.Conn) {
            go Send(c)
            Read(c)
        }(client)
    }
}

// 往 内网穿透服务器 发数据
func Send(c net.Conn) {
    for {
        select {
        case data := <- tcpToClientStream:
            fmt.Println("向内网发送数据")
            err := c.SetWriteDeadline(time.Now().Add(2 * time.Second))
            n, err := c.Write(append([]byte(strconv.Itoa(data.ConnId)+"\r\n"), data.Data...))
            if err != nil {
                if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                    continue
                }
                return
            }
            fmt.Println(n, err)
        }
    }
}

// 从 内网穿透服务器 读数据
func Read(c net.Conn) {
    defer fmt.Println("safrp client " + c.RemoteAddr().String() + " close ...")
    tBuf := BufPool.Get()
    buf := []byte{}
    if tBuf == nil {
        buf = make([]byte, 1024 * 100)
    } else {
        buf = tBuf.([]byte)
    }
    defer func() {
        BufPool.Put(buf)
    }()

    for {
        n, err := c.Read(buf)
        if err != nil {
            if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                continue
            }
            return
        }
        fmt.Println("从内网读取数据")
        tBuf := bytes.SplitN(buf[:n], []byte("\r\n"), 2)
        tId := 0
        for i := 0;i < len(tBuf[0]);i++ {
            if tBuf[0][i] != '\r' && tBuf[0][i] != '\n' {
                tId = tId * 10 + int(tBuf[0][i] - '0')
            }
        }
        fmt.Println("编号:", tId)
        tcpFromClientStream[tId].(chan TCPData) <- TCPData{
            ConnId: tId,
            Data:   tBuf[1],
        }
    }
}

package main

import (
    "fmt"
    "io"
    "net"
    "sync"
    "time"
)

var tcpToServerStream = make(chan TCPData, 1000)
var tcpFromServerStream = make(chan TCPData, 1000)
var closeConn = make(chan bool, 2)

var addr = "127.0.0.1:8001"

type TCPData struct {
    ConnId uint64
    Data []byte
}

var ConnPool = sync.Pool{}
var BufPool = sync.Pool{}


func main() {
    go proxyClient()
    go Client()
    select {}
}

func proxyClient() {
    fmt.Println("safrp client ...")
    for {
        conn, err := net.Dial("tcp", "127.0.0.1:8002")
        if err != nil {
            fmt.Println(err)
            time.Sleep(3 * time.Second)
            continue
        }
        fmt.Println("connect success ...")
        go Read(conn)
        Send(conn)
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
        if n == 0 {
            if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                 continue
            }
            return
        }
        fmt.Println(string(buf[:n]))
        tcpFromServerStream <- TCPData{
            ConnId: 0,
            Data:   buf[:n],
        }
    }
}

// 往 内网穿透服务器 发数据
func Send(c net.Conn) {
    for {
        select {
        case data := <- tcpFromServerStream:
            _, err := c.Write(data.Data)
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
            c, err := net.Dial("tcp", "127.0.0.1:81")
            if err != nil {
                return
            }
            go IntranetTransmitSend(c, data.Data)
            IntranetTransmitRead(c)
        }
    }
}

// 从 内网服务器 读数据
func IntranetTransmitRead(c net.Conn) {
    buf := make([]byte, 1024)
    for {
        err := c.SetReadDeadline(time.Now().Add(2 * time.Second))
        if err != nil {
            return
        }
        n, err := c.Read(buf)
        if err != nil {
            continue
        }
        fmt.Println(string(buf[:n]))
        tcpToServerStream <- TCPData{
            ConnId: 0,
            Data:   buf[:n],
        }
        return
    }
    return
}

// 往 内网服务器 发数据
func IntranetTransmitSend(c net.Conn, data []byte) {
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

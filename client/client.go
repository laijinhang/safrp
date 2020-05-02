package main

import (
    "bytes"
    "fmt"
    "io"
    "net"
    "strconv"
    "sync"
    "time"
)

var tcpToServerStream = make(chan TCPData, 1000)
var tcpFromServerStream = make(chan TCPData, 1000)
var closeConn = make(chan bool, 2)

var addr = "127.0.0.1:8001"

type TCPData struct {
    ConnId int
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
            c, err := net.Dial("tcp", "127.0.0.1:8003")
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
    buf := make([]byte, 1024)
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
    err := c.SetWriteDeadline(time.Now().Add(2 * time.Second))
    if err != nil {
        fmt.Println(err)
        return
    }
    _, err = c.Write(data)
    fmt.Println(err)
}

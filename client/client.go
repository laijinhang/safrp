package main

import (
    "net"
    "sync"
    "time"
)

var tcpToClientStream = make(chan TCPData, 1000)
var tcpFromClientStream = make(chan TCPData, 1000)

var addr = "192.168.1.2:8888"

type TCPData struct {
    ConnId uint64
    Data []byte
}

var ConnPool = sync.Pool{}
var BufPool = sync.Pool{}


func main() {
    proxyClient()
}

func proxyClient() {
    conn, err := net.Dial("tcp", ":10001")
    if err != nil {
        panic(err)
    }

    go Read(conn)
    Send(conn)
}

// 从 内网穿透服务器 读数据
func Read(c net.Conn) {
    tBuf := BufPool.Get()
    buf := []byte{}
    if tBuf == nil {
        buf = make([]byte, 1024 * 10)
    } else {
        buf = tBuf.([]byte)
    }
    for {
        err := c.SetReadDeadline(time.Now().Add(3 * time.Second))
        if err != nil {
            return
        }
        n, err := c.Read(buf)
        go func() {
            IntranetTransmitSend(buf)
            IntranetTransmitRead()

        }()

    }
}

// 往 内网穿透服务器 发数据
func Send(c net.Conn) {

}

// 从 内网服务器 读数据
func IntranetTransmitRead() ([]byte, error) {
    return nil, nil
}

// 往 内网服务器 发数据
func IntranetTransmitSend(buf []byte) ([]byte, error) {
    return nil, nil
}

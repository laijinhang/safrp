package main

import (
    "fmt"
    "net"
    "sync"
    "time"
)

var tcpToClientStream = make(chan TCPData, 1000)
var tcpFromClientStream = make(chan TCPData, 1000)

var ConnPool = sync.Pool{}
var BufPool = sync.Pool{}

func init() {
    for i := 1;i <= 1000;i++ {
        ConnPool.Put(i)
    }
}

type TCPData struct {
    ConnId uint64
    Data []byte
}

func main() {
    //proxyServer()
    go server()
    select {}
}

// 处理 穿透内外 网
func proxyServer() {

}

// 从 外网 接收数据
func ExtranetRead(c net.Conn) {
    tBuf := BufPool.Get()
    buf := []byte{}
    if tBuf == nil {
        buf = make([]byte, 1024 * 10)
    } else {
        buf = tBuf.([]byte)
    }

    err := c.SetReadDeadline(time.Now().Add(3 * time.Second))
    if err != nil {
        return
    }
    n, err := c.Read(buf)
    if err != nil {
        fmt.Println("error:", err)
        return
    }
}

// 往 外网 响应数据
func ExtranetSend() {
    for {
        c.SetReadDeadline(time.Now().Add(3 * time.Second))
        n, err := c.Read(buf1)
        if err != nil {
            return
        }
        c.SetWriteDeadline(time.Now().Add(3 * time.Second))
        n, err = client.Write(buf1[:n])

        if err != nil {
            return
        }
    }
}

// 处理外网服务
func server() {
    listen, err := net.Listen("tcp", ":81")
    if err != nil {
        panic(err)
    }
    for {
        client, err := listen.Accept()
        if err != nil {
            continue
        }
        go func(client net.Conn) {
            c, err := net.Dial("tcp", "192.168.1.3:8888")

            if err != nil {
                fmt.Println(err)
                return
            }
            defer c.Close()
            go func() {
            }()
            defer client.Close()
        }(client)
    }
}

// 往 内网穿透服务器 发数据
func Send(c net.Conn) {

}

// 从 内网穿透服务器 读数据
func Read(c net.Conn) {
    c.SetWriteDeadline(time.Now().Add(3 * time.Second))
    n, err = c.Write(buf[:n])
    if err != nil {
        return
    }
    n, err = c.Read(buf)
}

func a()  {
    for {
        listen, err := net.Listen("tcp", ":10001")
        if err != nil {
            panic(err)
        }
        for {
            client, err := listen.Accept()
            if err != nil {
                continue
            }
            // 读取
            go func(c net.Conn) {
                tBuf := BufPool.Get()
                buf := []byte{}
                if tBuf == nil {
                    buf = make([]byte, 1024 * 10)
                } else {
                    buf = tBuf.([]byte)
                }
                defer BufPool.Put(buf)

                for {
                    err := c.SetReadDeadline(time.Now().Add(3 * time.Second))
                    if err != nil {
                        break
                    }
                    //n, err := c.Read(buf)

                }
            }(client)
            // 发送

        }
    }
}


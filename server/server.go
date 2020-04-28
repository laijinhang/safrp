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
var DeadTime = make([]chan interface{}, 1001)

func init() {
    for i := 1;i <= 1000;i++ {
        ConnPool.Put(i)
    }
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
        fmt.Println(client, num)
        go func(c net.Conn, n int) {
            defer func() {
                ConnPool.Put(n)
                c.Close()
            }()
            //go ExtranetRead(c, n)
            ExtranetRead(c, n)
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

    for {
        err := c.SetReadDeadline(time.Now().Add(3 * time.Second))
        if err != nil {
            return
        }
        n, err := c.Read(buf)
        if err != nil {
            break
        }
        fmt.Println("从收到外网请求")
        tcpToClientStream <- TCPData{
            ConnId: number,
            Data:   buf[:n],
        }
    }
}

// 往 外网 响应数据
func ExtranetSend(c net.Conn, number int) {
    tBuf := BufPool.Get()
    buf := []byte{}
    if tBuf == nil {
        buf = make([]byte, 1024 * 10)
    } else {
        buf = tBuf.([]byte)
    }

    for {
        select {
        case data := <- tcpFromClientStream:
            err := c.SetWriteDeadline(time.Now().Add(3 * time.Second))
            if err != nil {
                continue
            }
            n, err := c.Write(data.Data)
            fmt.Println("向外网发送数据")
            fmt.Println(string(buf[:n]))
        }
    }
}

// 处理穿透内网服务
func server() {
    listen, err := net.Listen("tcp", ":8001")
    if err != nil {
        panic(err)
    }
    for {
        client, err := listen.Accept()
        if err != nil {
            continue
        }
        go func(c net.Conn) {
            //go Send(c)
            Send(c)
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
            // 转json
            //fmt.Println(string(data.Data))
            //buf, err := json.Marshal(data)
            //
            //if err != nil {
            //    fmt.Println(err)
            //    continue
            //}
            err := c.SetWriteDeadline(time.Now().Add(2 * time.Second))
            n, err := c.Write(data.Data)

            fmt.Println(n, err)
        }
    }
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
            continue
        }
        n, err := c.Read(buf)
        fmt.Println("送内网读取数据")
        tcpFromClientStream <- TCPData{
            ConnId: 0,
            Data:   buf[:n],
        }
    }
}

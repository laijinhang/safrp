package main

import (
    "fmt"
    "log"
    "net"
    "os"
    "strconv"
    "sync"
    "syscall"
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
            break
        }
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
    defer func() {
        BufPool.Put(buf)
    }()

    for {
        n, err := c.Read(buf)
        if err != nil {
            return
        }
        fmt.Println("从内网读取数据")
        fmt.Println(string(buf[:n]))
        tcpFromClientStream <- TCPData{
            ConnId: 0,
            Data:   buf[:n],
        }
    }
}

func isNetCloseError(err error) bool {
    netErr, ok := err.(net.Error)
    if !ok {
        return false
    }

    if netErr.Timeout() {
        log.Println("timeout")
        return true
    }

    opErr, ok := netErr.(*net.OpError)
    if !ok {
        return false
    }

    switch t := opErr.Err.(type) {
    case *net.DNSError:
        log.Printf("net.DNSError:%+v", t)
        return true
    case *os.SyscallError:
        log.Printf("os.SyscallError:%+v", t)
        if errno, ok := t.Err.(syscall.Errno); ok {
            switch errno {
            case syscall.ECONNREFUSED:
                log.Println("connect refused")
                return true
            case syscall.ETIMEDOUT:
                log.Println("timeout")
                return true
            }
        }
    }

    return false
}

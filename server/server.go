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
    Password string
    ServerIP string
    ServerPort string
    SafrpIP string
    SafrpPort string
}

type NumberPool struct {
    val []interface{}
    mutex sync.Mutex
}

func (n *NumberPool)Put(x interface{}) {
    n.mutex.Lock()
    defer n.mutex.Unlock()
    n.val = append(n.val, x)
}


func (n *NumberPool)Get() interface{} {
    n.mutex.Lock()
    defer n.mutex.Unlock()
    if len(n.val) == 0 {
        return nil
    }
    a := n.val[0]
    n.val = n.val[1:]
    return a
}

var conf Config
var BufSize = 1024 * 8

var tcpToClientStream = make(chan TCPData, 1000)
var tcpFromClientStream [1001]interface{}

var ConnPool = NumberPool{}
var BufPool = sync.Pool{New: func() interface{} {return make([]byte, BufSize)}}
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
    temp, _ =cfg.Section("safrp").GetKey("port")
    conf.SafrpPort = temp.String()
    temp, _ =cfg.Section("").GetKey("password")
    conf.Password = temp.String()
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
    listen, err := net.Listen("tcp", conf.ServerIP + ":" + conf.ServerPort)
    fmt.Println("listen :" + conf.ServerIP + ":" + conf.ServerPort + " ...")
    if err != nil {
        panic(err)
    }
    for {
        client, err := listen.Accept()
        if err != nil {
            return
        }
        go func(c net.Conn) {
            defer func() {
                for err := recover();err != nil;err = recover(){
                }
            }()
            defer c.Close()
            num := -1
            for c := 0;num == -1;c++ {
                tNum := ConnPool.Get()
                if tNum != nil {
                    num = tNum.(int)
                    break
                } else {
                    time.Sleep(50 * time.Millisecond)
                }
                if c == 20 {
                    return
                }
            }
            fmt.Println(client.RemoteAddr(), num)
            defer func() {
               ConnPool.Put(num)
               c.Close()
            }()
            go ExtranetRead(c, num)
            ExtranetSend(c, num)
        }(client)
    }
}

// 从 外网 接收数据
func ExtranetRead(c net.Conn, number int) {
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
        fmt.Println(string(buf[:n]))
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
            if time.Now().Unix() - BeginTime >= int64(6) {
                c.Close()
                return
            }
        }
    }
}

// 处理穿透内网服务
func server() {
    listen, err := net.Listen("tcp", ":"+ conf.SafrpPort)
    fmt.Println("safrp server listen :" + conf.SafrpPort + " ...")
    if err != nil {
        panic(err)
    }
    for {
        client, err := listen.Accept()
        if err != nil {
            fmt.Println(err)
            continue
        }
        go func(c net.Conn) {
            defer func() {
                for err := recover();err != nil;err = recover(){
                }
            }()
            fmt.Println("frp client尝试建立连接...")
            buf := BufPool.Get().([]byte)
            err := c.SetReadDeadline(time.Now().Add(3 * time.Second))
            if err != nil {
                fmt.Println("password error...")
                return
            }
            n, err := c.Read(buf)
            if err != nil || string(buf[:n]) != conf.Password {
                fmt.Println("password error...")
                return
            }
            defer fmt.Println("safrp client " + c.RemoteAddr().String() + " close ...")
            err = c.SetWriteDeadline(time.Now().Add(3 * time.Second))
            n, err = c.Write([]byte("connect success ..."))
            if err != nil {
                return
            }

            fmt.Println("safrp client ", client.RemoteAddr(), "connect success ...")
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
            err := c.SetWriteDeadline(time.Now().Add(2 * time.Second))
            if err != nil {
                return
            }
            _, err = c.Write(append([]byte(strconv.Itoa(data.ConnId)+"\r\n"), data.Data...))
            if err != nil {
                if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                    continue
                }
                return
            }
        }
    }
}

// 从 内网穿透服务器 读数据
func Read(c net.Conn) {
    ReadStream(c)
}

func ReadStream(c net.Conn) {
    defer func() {
        for err := recover();err != nil;err = recover(){
        }
        c.Close()
    }()
    streamBuf := make([]byte, 1024 * 1024 * 8)
    streamData := make(chan []byte, 1024)
    closeConn := make(chan bool, 3)
    var n int
    var err error
    go func() {
        for {
            select {
            case data := <- streamData:
                tData := bytes.Split(data, []byte("date_end;"))
                for len(tData) != 0 {
                    if bytes.HasSuffix(tData[0], []byte("data_end;")) { // 数据是完整的
                        if len(tData[0]) == 10 && bytes.HasSuffix(tData[0], []byte("data_end;")) {
                            if len(tData) == 1 {
                                return
                            }
                            tData = tData[1:]
                        }

                        tData[0] = tData[0][:len(tData[0])-9]
                        tBuf := bytes.SplitN(tData[0], []byte("\r\n"), 2)
                        tId := 0
                        for i := 0; i < len(tBuf[0]); i++ {
                            if tBuf[0][i] != '\r' && tBuf[0][i] != '\n' {
                                tId = tId*10 + int(tBuf[0][i]-'0')
                            }
                        }
                        fmt.Println("编号id：", tId)
                        if tId > 1000 || tId < 0 {
                            continue
                        }
                        tcpFromClientStream[tId].(chan TCPData) <- TCPData{
                            ConnId: tId,
                            Data:   tBuf[1],
                        }
                    } else {
                        select {
                        case data = <- streamData:
                            tData = bytes.SplitN(append(tData[0], data...), []byte("date_end;"), 2)
                        case <-closeConn:
                            return
                        }
                    }
                }
            case <-closeConn:
                return
            }
        }
    }()

    buf := BufPool.Get()
    defer func() {
        BufPool.Put(buf)
    }()
    for {
        err = c.SetReadDeadline(time.Now().Add(1 * time.Second))
        if err != nil {
            closeConn <- true
            return
        }
        n, err = c.Read(streamBuf)
        if err != nil {
            if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                continue
            }
            closeConn <- true
            return
        }
        streamData <- streamBuf[:n]
    }
}

package main

import (
    "bytes"
    "fmt"
    "gopkg.in/ini.v1"
    "io"
    "log"
    "net"
    "safrp/common"
    "strconv"
    "sync"
    "sync/atomic"
    "time"
)

type Config struct {
    Password string
    ServerIP string
    ServerTCPPort string
    ServerUDPPort string
    SafrpIP string
    SafrpPort string
}

type NumberPool struct {
    numberArr []uint64
    number uint64
    maxVal uint64
    add uint64
}

func New(val, add uint64) *NumberPool {
    return &NumberPool{
        numberArr:make([]uint64, val+1),
        number: 1,
        maxVal: val,
        add:    add,
    }
}

func (n *NumberPool)Get() (uint64, bool) {
    num := 0
    for i := atomic.LoadUint64(&n.number);;i = atomic.AddUint64(&n.number, n.add) {
        atomic.CompareAndSwapUint64(&n.number, n.maxVal, 1)
        num++
        if num / int(n.maxVal) >= 3 {
            return 0, false
        }
        if i > n.maxVal {
            i = 1
        }
        if atomic.CompareAndSwapUint64(&n.numberArr[i], 0, 1) {
            return i, true
        }
    }
    return 0, false
}

func (n *NumberPool)Put(v int) {
    atomic.CompareAndSwapUint64(&n.numberArr[v], 1, 0)
}

var conf Config
var BufSize = 1024 * 10 * 8
var TCPDataEnd = []byte("data_end;")

var tcpToClientStream = make(chan TCPData, 10000)
var tcpFromClientStream [2001]interface{}
var maxNum = 2000

var ConnPool = New(2000, 1)
var BufPool = sync.Pool{New: func() interface{} {return make([]byte, BufSize)}}
var DeadTime = make([]chan interface{}, 2001)

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
    temp, _ =cfg.Section("server").GetKey("tcp_port")
    conf.ServerTCPPort = temp.String()
    temp, _ =cfg.Section("server").GetKey("udp_port")
    conf.ServerUDPPort = temp.String()
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
    go run(proxyTCPServer)  // 处理TCP外网请求，短连接服务
    go run(proxyUDPServer)  // 处理UDP外网请求
    go run(server)          // 处理TCP外网请求，短连接服务
    select {}
}

func run(server func()) {
    wg := sync.WaitGroup{}
    for {
        wg.Add(1)
        go func() {
            defer wg.Done()
            defer func() {
                for err := recover(); err != nil; err = recover() {
                    fmt.Println(err)
                }
            }()

            server()
        }()
        wg.Wait()
        time.Sleep(time.Second)
    }
}

// 处理 TCP外网 请求
func proxyTCPServer() {
    listen, err := net.Listen("tcp", conf.ServerIP + ":" + conf.ServerTCPPort)
    fmt.Println("listen :" + conf.ServerIP + ":" + conf.ServerTCPPort + " ...")
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
                    fmt.Println(err)
                }
            }()
            defer c.Close()
            num := -1
            for c := 0;num == -1;c++ {
                tNum, ok := ConnPool.Get()
                if ok {
                    num = int(tNum)
                    break
                } else {
                    time.Sleep(50 * time.Millisecond)
                }
                if c == 20 {
                    return
                }
            }

            tcpFromClientStream[num] = make(chan TCPData, 30)
            fmt.Println("请求：", client.RemoteAddr(), num)
            defer func() {
                tcpToClientStream <- TCPData{
                    ConnId: num,
                    Data:   common.SafrpTCPPackage([]byte(""), TCPDataEnd),
                }
                ConnPool.Put(num)
                c.Close()
                close(tcpFromClientStream[num].(chan TCPData))
            }()
            go ExtranetTCPRead(c, num)
            ExtranetTCPSend(c, num)
        }(client)
    }
}

// 处理 UDP外网 请求
func proxyUDPServer() {
    //wg := sync.WaitGroup{}
    //for {
    //    wg.Add(1)
    //    go func() {
    //        defer wg.Done()
    //        defer func() {
    //            for v := recover();v != nil;v = recover() {
    //                fmt.Println(v)
    //            }
    //        }()
    //        udpAddr, err := net.ResolveUDPAddr("udp", conf.ServerIP+":"+conf.ServerUDPPort)
    //        if err != nil {
    //            panic(err)
    //        }
    //        for {
    //            conn, err := net.ListenUDP("udp", udpAddr)
    //            if err != nil {
    //                fmt.Println(err)
    //                continue
    //            }
    //            go func(c net.Conn) {
    //                // 处理连接进来的读操作
    //                // 处理连接进来的写操作
    //            }(conn)
    //        }
    //    }()
    //    wg.Wait()
    //}
}

// 从 外网 接收TCP数据
func ExtranetTCPRead(c net.Conn, number int) {
    buf := BufPool.Get().([]byte)
    defer func() {
        BufPool.Put(buf)
    }()

    deadTime := time.Now().Unix()
    for {
        err := c.SetReadDeadline(time.Now().Add(1 * time.Second))
        if err != nil {
            return
        }
        n, err := c.Read(buf)
        if err != nil {
            if _, ok := err.(net.Error); ok && err == io.EOF {
                if time.Now().Unix() - deadTime > 3 {
                    return
                }
                continue
            }
            return
        }
        fmt.Println(string(buf[:n]))
        //// 如果不是HTTP协议，直接过滤掉
        //_, protocol := common.TCPApplicationLayerProtocolIdentification(buf[:n])
        //if protocol == "TCP" {
        //    fmt.Println("TCP")
        //    return
        //}
        //fmt.Println(protocol)
        deadTime = time.Now().Unix()
        tcpToClientStream <- TCPData{
            ConnId: number,
            Data:   buf[:n],
        }
    }
}

// 往 外网 响应TCP数据
func ExtranetTCPSend(c net.Conn, number int) {
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
                if _, ok := err.(net.Error); ok &&  err == io.EOF {
                    continue
                }
                return
            }
            BeginTime = time.Now().Unix()
        default:
            if time.Now().Unix() - BeginTime >= int64(6) {
                c.Close()
                return
            }
        }
    }
}

// 从 外网 接收UDP数据
func ExtranetUDPRead(c net.Conn, number int) {

}

// 往 外网 响应UDP数据
func ExtranetUDPSend(c net.Conn, number int) {

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
                    fmt.Println(err)
                }
            }()
            defer c.Close()
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
            _, err = c.Write(append([]byte(c.RemoteAddr().String() + " "+strconv.Itoa(data.ConnId)+"\r\n"), data.Data...))
            if err != nil {
                if _, ok := err.(net.Error); ok && err == io.EOF {
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
            fmt.Println(err)
        }
    }()
    defer c.Close()
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
                           continue
                       }

                       tData[0] = tData[0][:len(tData[0])-9]
                       tBuf := bytes.SplitN(tData[0], []byte("\r\n"), 2)
                       tId := 0
                       for i := 0; i < len(tBuf[0]); i++ {
                           if tBuf[0][i] != '\r' && tBuf[0][i] != '\n' {
                               tId = tId*10 + int(tBuf[0][i]-'0')
                           }
                       }
                       if tId > maxNum || tId < 0 {
                           continue
                       }
                       fmt.Println("编号id：", tId)
                       go func(tId int, data TCPData) {
                           defer func() {
                               for err := recover();err != nil;err = recover(){
                               }
                           }()
                           if atomic.LoadUint64(&ConnPool.numberArr[tId]) == 1 {
                               tcpFromClientStream[tId].(chan TCPData) <- data
                           }

                       }(tId, TCPData{
                           ConnId: tId,
                           Data:   bytes.TrimSuffix(tBuf[1], []byte("date_end;"))})
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

    buf := BufPool.Get().([]byte)
    defer func() {
       BufPool.Put(buf)
    }()
    for {
        err = c.SetReadDeadline(time.Now().Add(1 * time.Second))
        if err != nil {
            fmt.Println(err)
            closeConn <- true
            return
        }
        n, err = c.Read(buf)
        if err != nil {
            if neterr, ok := err.(net.Error);ok && (neterr.Timeout() || err == io.EOF) {
                continue
            }
            closeConn <- true
            return
        }
        streamData <- buf[:n]
    }
}

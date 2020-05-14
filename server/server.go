package main

import (
    "bytes"
    "fmt"
    "github.com/sirupsen/logrus"
    "gopkg.in/ini.v1"
    "io"
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
    logrus.Infoln(v)
    logrus.Infoln(atomic.CompareAndSwapUint64(&n.numberArr[v], 1, 0))
}

var conf Config
var BufSize = 1024 * 1024 * 8
var TCPDataEnd = []byte{'<','e','>'}

var tcpToClientStream = make(chan TCPData, 10000)
var tcpFromClientStream [2001]interface{}
var maxNum = 2000

var ConnPool = New(2000, 1)
var BufPool = sync.Pool{New: func() interface{} {return make([]byte, BufSize)}}
var DeadTime = make([]chan interface{}, 2001)

func init() {
    for i := 1;i <= 1000;i++ {
        tcpFromClientStream[i] = make(chan TCPData, 10)
    }
    cfg, err := ini.Load("./safrp.ini")
    if err != nil {
        logrus.Panicln("Fail to read file: ", err)
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


    logrus.SetLevel(logrus.TraceLevel)
    logrus.SetFormatter(&logrus.TextFormatter{
        ForceColors:               true,
        FullTimestamp:             true,
        TimestampFormat:           "2006-01-02 15:04:05",
        DisableSorting:            false,
        SortingFunc:               nil,
        DisableLevelTruncation:    true,
        QuoteEmptyFields:          false,
        FieldMap:                  nil,
        CallerPrettyfier:          nil,
    })
    logrus.SetReportCaller(true)
    logrus.SetLevel(logrus.PanicLevel)
}

type TCPData struct {
    ConnId int
    Data []byte
}

func main() {
    go common.Run(proxyTCPServer)  // 处理TCP外网请求，短连接服务
    go common.Run(proxyUDPServer)  // 处理UDP外网请求
    go common.Run(SafrpTCPServer)  // 处理TCP外网请求，短连接服务
    select {}
}

// 处理 TCP外网 请求
func proxyTCPServer() {
    listen, err := net.Listen("tcp", conf.ServerIP + ":" + conf.ServerTCPPort)
    logrus.Infoln("listen :" + conf.ServerIP + ":" + conf.ServerTCPPort + " ...")
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
                for p := recover();p != nil;p = recover(){
                    logrus.Errorln(p)
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
            logrus.Infoln("请求：", client.RemoteAddr(), num)
            defer func() {
                tcpToClientStream <- TCPData{
                    ConnId: num,
                    Data:   []byte(""),
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
    wg := sync.WaitGroup{}
    for {
       wg.Add(1)
       go func() {
           defer wg.Done()
           defer func() {
               for p := recover();p != nil;p = recover() {
                   logrus.Println(p)
               }
               time.Sleep(3 * time.Second)  // 缓慢重启
           }()
           udpAddr, err := net.ResolveUDPAddr("udp", conf.ServerIP+":"+conf.ServerUDPPort)
           if err != nil {
               logrus.Errorln(err)
               
           }
           for {
               conn, err := net.ListenUDP("udp", udpAddr)
               if err != nil {
                   fmt.Println(err)
                   continue
               }
               go func(c net.Conn) {
                   // 处理连接进来的读操作
                   // 处理连接进来的写操作
               }(conn)
           }
       }()
       wg.Wait()
    }
}

// 从 外网 接收TCP数据
func ExtranetTCPRead(c net.Conn, number int) {
    buf := BufPool.Get().([]byte)
    defer func() {
        BufPool.Put(buf)
    }()

    deadTime := time.Now().Unix()
    for {
        err := c.SetReadDeadline(time.Now().Add(12 * time.Second))
        if err != nil {
            logrus.Errorln(err)
            return
        }
        n, err := c.Read(buf)
        if err != nil {
            logrus.Errorln(err)
            if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                if time.Now().Unix() - deadTime > 5 {
                    return
                }
                continue
            }
            return
        }
        deadTime = time.Now().Unix()

        tcpToClientStream <- TCPData{
            ConnId: number,
            Data:   append([]byte(c.RemoteAddr().String() + " " + strconv.Itoa(number) + "\r\n"), buf[:n]...),
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
            logrus.Infoln(string(data.Data))
            err := c.SetWriteDeadline(time.Now().Add(3 * time.Second))

            if err != nil {
                logrus.Errorln(err)
                continue
            }
            _, err = c.Write(data.Data)
            if err != nil {
                logrus.Errorln(err)
                if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
                    continue
                }
                return
            }
            BeginTime = time.Now().Unix()
            return
        default:
            if time.Now().Unix() - BeginTime >= int64(12) {
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
func SafrpTCPServer() {
    listen, err := net.Listen("tcp", ":"+ conf.SafrpPort)
    logrus.Infoln("safrp server listen :" + conf.SafrpPort + " ...")
    if err != nil {
        logrus.Panicln(err)
    }
    for {
        client, err := listen.Accept()
        if err != nil {
            logrus.Errorln(err)
            continue
        }
        go func(c net.Conn) {
            defer func() {
                for p := recover();p != nil;p = recover(){
                    logrus.Panicln(p)
                }
            }()
            defer c.Close()
            logrus.Infoln("frp client尝试建立连接...")
            buf := BufPool.Get().([]byte)
            err := c.SetReadDeadline(time.Now().Add(3 * time.Second))
            if err != nil {
                logrus.Infoln("password error...")
                return
            }
            n, err := c.Read(buf)
            if err != nil || string(buf[:n]) != conf.Password {
                logrus.Infoln("password error...")
                return
            }
            defer logrus.Infoln("safrp client " + c.RemoteAddr().String() + " close ...")
            err = c.SetWriteDeadline(time.Now().Add(3 * time.Second))
            n, err = c.Write([]byte("connect success ..."))
            if err != nil {
                logrus.Errorln(err)
                return
            }

            logrus.Infoln("safrp client ", client.RemoteAddr(), "connect success ...")

            var wg sync.WaitGroup

            var steamChan = make(chan []byte, 1000)
            var dataChan = make(chan []byte, 1000)

            wg.Add(3)
            go func() {
                defer wg.Done()
                common.Run(func() {
                    Send(c)
                })
            }()
            go func() {
                defer wg.Done()
                common.Run(func() {
                    common.TCPSafrpStream(steamChan, dataChan, TCPDataEnd)
                })
            }()
            go func() {
                defer wg.Done()
                go common.Run(func() {
                    Read(c, steamChan, dataChan)
                })
            }()
            wg.Wait()
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
                logrus.Infoln(err)
                return
            }
            _, err = c.Write(common.SafrpTCPPackage(data.Data, TCPDataEnd))
            if err != nil {
                logrus.Errorln(err)
                if _, ok := err.(net.Error); ok && err == io.EOF {
                    continue
                }
                return
            }
        }
    }
}

// 从 内网穿透服务器 读数据
func Read(c net.Conn, steamChan, dataChan chan []byte) {
    defer func() {
        for p := recover();p != nil;p = recover() {
            logrus.Panicln(p)
        }
    }()
    closeConn := make(chan bool, 3)
    var n int
    var err error

    buf := BufPool.Get().([]byte)
    defer func() {
        c.Close()
        BufPool.Put(buf)
    }()
    go func() {
        defer func() {
            for p := recover(); p != nil; p = recover() {
                logrus.Println(p)
            }
        }()
        for {
            select {
            case buf := <-dataChan:
                logrus.Infoln(string(buf))
                if len(buf) == 0 {  // 心跳包
                    continue
                }
                tBuf := bytes.SplitN(buf, []byte("\r\n"), 2)
                tId := 0
                for i := 0; i < len(tBuf[0]); i++ {
                    if tBuf[0][i] != '\r' && tBuf[0][i] != '\n' {
                        tId = tId*10 + int(tBuf[0][i]-'0')
                    }
                }
                logrus.Infoln(tId)
                logrus.Infoln(atomic.LoadUint64(&ConnPool.numberArr[tId]))
                if atomic.LoadUint64(&ConnPool.numberArr[tId]) == 1 {
                    logrus.Infoln("响应,id:", tId, string(tBuf[1]))
                    tcpFromClientStream[tId].(chan TCPData) <- TCPData{
                        ConnId: tId,
                        Data:   tBuf[1],
                    }
                }
            }
        }
    }()
    for {
        err = c.SetReadDeadline(time.Now().Add(5 * 60 * time.Second))
        if err != nil {
            logrus.Errorln(err)
            closeConn <- true
            return
        }
        n, err = c.Read(buf)
        if err != nil {
            logrus.Errorln(err)
            if neterr, ok := err.(net.Error);ok && (neterr.Timeout() || err == io.EOF) {
                continue
            }
            closeConn <- true
            return
        }

        logrus.Infoln(n, string(buf[:n]))
        steamChan <- buf[:n]
    }
}

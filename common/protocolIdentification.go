package common

import (
	"bytes"
	"fmt"
	"net"
	"sync"
)

var HTTP = sync.Map{}
var WebSocket = sync.Map{}

func init()  {
	WebSocket.Store("Upgrade", "websocket")
	WebSocket.Store("Connection", "Upgrade")
}

func main() {
	listen, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go func(c net.Conn) {
			buf := make([]byte, 1024)
			n, err := c.Read(buf)
			if err != nil {
				return
			}
			data, protocol := TCPApplicationLayerProtocolIdentification(buf[:n])
			fmt.Println(data, protocol)
		}(conn)
	}
}

/*
识别基于TCP的应用层协议类型
支持识别：
1. TCP
2. HTTP
3. WebSocket
*/
func TCPApplicationLayerProtocolIdentification(d []byte) (data map[string]string, protocol string) {
	data = make(map[string]string)
	bs := bytes.Split(d, []byte("\r\n"))
	if len(bs) < 4 {
		return nil, "TCP"
	}

	protocol = "HTTP"
	h := bytes.Split(bs[0], []byte(" "))
	if len(h) != 3 {
		return nil, "TCP"
	}
	data["method"] = string(h[0])
	data["url"] = string(h[1])
	data["protocol"] = string(h[2])

	if len(data["protocol"]) < 4 || data["protocol"][:4] != "HTTP" {
		return
	}
	for i := 1;i < len(bs);i++ {
		tb := bytes.SplitN(bs[i], []byte(": "), 2)
		if len(tb) < 2 {
			continue
		}
		data[string(tb[0])] = string(tb[1])
	}

	protocol = "WebSocket"
	WebSocket.Range(func(key, value interface{}) bool {
		if v, ok := data[key.(string)];!ok || v != value {
			protocol = "HTTP"
			return false
		}
		return true
	})

	return
}

/*
识别基于UDP的应用层协议类型
1. UDP
2. QUIC
*/
func UDPApplicationLayerProtocolIdentification(d []byte) (data map[string]string, protocol string) {
	return
}

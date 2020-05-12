package common

import (
	"net"
	"time"
)

/*
一次加载完整的HTTP请求数据
 */
func HTTPReq(c net.Conn, buf []byte) (req , other []byte) {
	// buf是HTTP请求头已开始部分
	tempBuf := make([]byte, 1024 * 8)
	for {
		err := c.SetReadDeadline(time.Now().Add(time.Second))
		if err != nil {
			return
		}
		n, err := c.Read(tempBuf)
		if n < 1024 * 8 {
		}
	}
	return
}

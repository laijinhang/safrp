package common

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"strconv"
	"time"
)

type HTTPClient struct {
	Addr string
	Number uint64
	Conn net.Conn
	close bool
	DeadTime int
	ReadDeadTime int
	WriteDeadTime int
}

/*
创建连接
 */
func (c *HTTPClient)Dail(address string) (err error) {
	c.Conn, err = net.Dial("tcp", address)
	if err != nil {
		c.close = true
	} else {
		c.close = false
	}
	return
}

/*
发起一次HTTP请求
 */
func (c *HTTPClient)Write(data []byte) error {
	defer func() {
		for p := recover();p != nil;p = recover() {
			logrus.Println("panic", p)
		}
		c.close = true
	}()
	c.setDeadTime()
	_, err := c.Conn.Write(data)
	return err
}

/*
读取一次完整的http响应数据
 */
func (c *HTTPClient)Read() ([]byte, error) {
	defer func() {
		for p := recover();p != nil;p = recover() {
			logrus.Println("panic", p)
		}
		c.close = true
	}()
	resp := []byte{}
	respHeader := make(map[string]string)
	buf := make([]byte, 10240)

	for {
		c.setDeadTime()
		n, err := c.Conn.Read(buf)
		if err != nil {
			logrus.Errorln(err)
			if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
				continue
			}
			return nil, err
		}

		temp := bytes.SplitN(buf[:n], []byte("\r\n\r\n"), 2)
		tempH := bytes.Split(temp[0], []byte("\r\n"))
		for i := 1; i < len(tempH); i++ {
			t := bytes.Split(tempH[i], []byte(": "))
			respHeader[string(t[0])] = string(t[1])
		}
		cl, _ := strconv.Atoi(respHeader["Content-Length"])
		cl += len(temp[0])
		resp = append(resp, buf[:n]...)

		for cl > len(resp) {
			c.setDeadTime()
			n, err := c.Conn.Read(buf)
			if err != nil {
				if neterr, ok := err.(net.Error); ok && (neterr.Timeout() || err == io.EOF) {
					continue
				}
				return nil, err
			}
			resp = append(resp, buf[:n]...)
		}
		break
	}
	return resp, nil
}

/*
检测该连接是否已经断开
 */
func (c *HTTPClient)ConnIsClose() bool {
	if c.Conn == nil {
		return true
	}
	if c.close == true {
		return true
	}
}

func (c *HTTPClient)setDeadTime() {
	if c.DeadTime > 0 {
		c.Conn.SetDeadline(time.Now().Add(time.Duration(c.DeadTime)))
	}
	if c.ReadDeadTime > 0 {
		c.Conn.SetReadDeadline(time.Now().Add(time.Duration(c.ReadDeadTime)))
	}
	if c.WriteDeadTime > 0 {
		c.Conn.SetWriteDeadline(time.Now().Add(time.Duration(c.WriteDeadTime)))
	}
}

func (c *HTTPClient)Close() bool {
	return c.close
}

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

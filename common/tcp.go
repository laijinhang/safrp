package common

import (
	"bytes"
	"github.com/sirupsen/logrus"
)

/*
解析safrp服务端与safrp客户端之间的TCP数据流解析
三种数据格式：
1. 普通数据
2. 心跳包
3. safrp服务端通知safrp客户端连接断开
 */
func TCPSafrpStream(stream, data chan []byte, end []byte) {
	buf := make([]byte, 0)
	for {
		select {
		case d := <- stream:
			logrus.Infoln(len(d))
			buf = append(buf, d...)
			for i := bytes.Index(buf, end);i != -1;i = bytes.Index(buf, end) {
				tempBuf := bytes.Split(buf, end)
				l := len(tempBuf) - 1
				if bytes.HasSuffix(buf, end) {
					buf = []byte{}
					l++
				} else {
					buf = tempBuf[len(tempBuf)-1]
				}
				for i := 0;i < l;i++ {
					if len(tempBuf[i]) == 0 {
						continue
					}
					go func(buf []byte) {
						logrus.Infoln("开始发送")
						data <- buf
						logrus.Infoln("发送结束")
					}(tempBuf[i])
					logrus.Infoln(string(tempBuf[i]))
				}
			}
		}
	}
}

/*
TCP数据结尾部分加上结束符
 */
func SafrpTCPPackage(data, end []byte) []byte {
	return append(data, end...)
}

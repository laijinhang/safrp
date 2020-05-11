package common

import (
	"bytes"
	"strconv"
)

/*
检查HTTP协议数据是否完整
 */
func verifyHTTPProtocolData(req []byte) bool {
	tempD := bytes.Split(req, []byte("\r\n\r\n"))
	if len(tempD) != 2 {
		return false
	}
	// 检验第一个行的请求头是否正确
	header := bytes.Split(tempD[0], []byte{'\r','\n'})
	if len(header) == 0 || len(bytes.Split(header[0], []byte{' '})) != 3 {
		return false
	}
	// 读取Content-Length
	headerM := make(map[string]string)
	for i := 1;i < len(header);i++ {
		temp := bytes.Split(header[i], []byte{':', ' '})
		if len(temp) != 2 {
			return false
		}
		headerM[string(temp[0])] = string(temp[1])
	}
	cl, ok := headerM["Content-Length"]
	if ok && len(tempD) != 2 {
		return false
	}
	if !ok && len(tempD) != 1 {
		return false
	}
	if !ok {
		return true
	}
	l, _ := strconv.Atoi(cl)
	return l == len(tempD[1])
}

package common

import "strconv"

// 打包
func Pack(number uint64, data []byte) []byte {
	s := strconv.FormatInt(int64(number), 10)
	return append([]byte(s + " " + strconv.Itoa(len(s) +  2 + len(data)) + " "), data...)
}
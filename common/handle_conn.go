package common

import "net"

func HandleConn(c1, c2 net.Conn) {
	go handleConn(c1, c2)
	go handleConn(c2, c1)
}

func handleConn(c1, c2 net.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := c1.Read(buf)
		if err != nil {
			break
		}
		_, err = c2.Write(buf[:n])
		if err != nil {
			break
		}
	}
}

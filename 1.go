package main

import (
	"fmt"
	"net"
	"net/http"
	"safrp/common"
)

func main() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintln(writer, "123456")
	})
	http.ListenAndServe(":9000", nil)
	return
	listen, err := net.Listen("tcp", ":10000")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go func(conn net.Conn) {
			buf := make([]byte, 10240)
			var c common.HTTPClient
			c.Dail(":81")
			n, err := conn.Read(buf)
			//fmt.Println(string(buf[:n]), err)
			c.Write(buf[:n])
			buf, err = c.Read()
			fmt.Println(err)
			conn.Write(buf)
			fmt.Println(len(buf))
			//fmt.Println(string(buf))
		}(conn)
	}
}

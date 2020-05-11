package main

import (
	"fmt"
	"safrp/common"
)

func main() {
	steam, data := make(chan []byte, 1000), make(chan []byte, 1000)
	go common.TCPStream(steam, data, []byte("end;"))
	go func() {
		for {
			select {
			case d := <-data:
				fmt.Println(string(d))
			}
		}
	}()
	var buf string
	for {
		fmt.Scan(&buf)
		go func(b []byte) {
			steam <- b
		}([]byte(buf))
	}
}

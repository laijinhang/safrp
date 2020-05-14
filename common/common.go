package common

import (
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

func Run(server func()) {
	wg := sync.WaitGroup{}
	for {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				for p := recover();p != nil;p = recover() {
					logrus.Println("panic:", p)
				}
			}()

			server()
		}()
		wg.Wait()
		time.Sleep(time.Second)
	}
}
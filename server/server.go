package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"safrp/common"
)

type Config struct {
	IP              string
	ExtranetPort    string
	ExtranetConnNum int
	ServerPort      string
	ClientPort		string
	Protocol        string
	Password        string
}

var confs []Config

/**
 * 加载safrp服务端配置文件
 * @param		nil
 * @return		nil
 * func loadConfig();
 */
func loadConfig() {
	cfg, err := ini.Load("./safrp.ini")
	if err != nil {
		logrus.Fatal("Fail to read file: ", err)
	}

	for i := 1; len(cfg.Section(fmt.Sprintf("pipe%d", i)).Keys()) != 0; i++ {
		pipeConf := cfg.Section(fmt.Sprintf("pipe%d", i))
		confs = append(confs, Config{
			IP:           pipeConf.Key("ip").String(),
			ExtranetPort: pipeConf.Key("extranet_port").String(),
			ExtranetConnNum: func(v int, e error) int {
				if e != nil {
					panic(e)
				}
				return v
			}(pipeConf.Key("extranet_conn_num").Int()),
			ServerPort: pipeConf.Key("server_port").String(),
			ClientPort: pipeConf.Key("client_port").String(),
			Protocol:   pipeConf.Key("protocol").String(),
			Password:   pipeConf.Key("password").String(),
		})
	}
}

func main() {
	loadConfig()
	for i := 0;i < len(confs);i++ {
		common.Run(func() {
			userClient := common.NewTcpServer(confs[i].IP, confs[i].ExtranetPort, confs[i].ExtranetConnNum)
			safrpServer := common.NewTcpServer(confs[i].IP, confs[i].ServerPort, confs[i].ExtranetConnNum)
			safrpClient := common.NewTcpServer(confs[i].IP, confs[i].ClientPort, confs[i].ExtranetConnNum)

			go userClient.Run()
			go safrpServer.Run()
			go safrpClient.Run()

			s := <-safrpServer.GetConn()
			for {
				select {
				case c1 := <- userClient.GetConn():
					s.Write([]byte(fmt.Sprintf("%v", c1)))
					c2 := <-safrpClient.GetConn()
					common.HandleConn(c1, c2)
				}
			}

			select {}
		})
	}
}



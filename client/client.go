package main

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"log"
	"net"
	"safrp/common"
)

type Config struct {
	ServerIP   string
	ServerPort string
	ClientPort string
	Password   string
	HTTPIP     string
	HTTPPort   string
	Protocol   string
	PipeNum    int
}

var conf Config

/**
 * 加载safrp服务端配置文件
 * @param		nil
 * @return		nil
 * func loadConfig();
 */
func loadConfig() {
	cfg, err := ini.Load("./safrp.ini")
	if err != nil {
		log.Fatal("Fail to read file: ", err)
	}
	serverIni := cfg.Section("server")
	conf.ServerIP = serverIni.Key("ip").String()
	conf.ServerPort = serverIni.Key("serverPort").String()
	conf.Password = serverIni.Key("password").String()
	conf.ClientPort = serverIni.Key("clientPort").String()

	proxyIni := cfg.Section("proxy")
	conf.HTTPIP = proxyIni.Key("ip").String()
	conf.HTTPPort = proxyIni.Key("port").String()
	conf.Protocol = proxyIni.Key("protocol").String()

	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: true,
	})
	logrus.SetReportCaller(true)

	logrus.Infoln("load safrp.ini ...")
	logrus.Infoln("server-ip:", conf.ServerIP)
	logrus.Infoln("server-port:", conf.ServerPort)
	logrus.Infoln("client-port:", conf.ClientPort)
	logrus.Infoln("server-password:", conf.Password)
	logrus.Infoln("http-ip:", conf.HTTPIP)
	logrus.Infoln("http-port:", conf.HTTPPort)
	logrus.Infoln()

	logrus.SetLevel(logrus.PanicLevel)
}

func main() {
	loadConfig()
	common.Run(func() {
		log := logrus.New()
		log.SetLevel(logrus.TraceLevel)
		log.SetFormatter(&logrus.TextFormatter{
			ForceColors:            true,
			FullTimestamp:          true,
			TimestampFormat:        "2006-01-02 15:04:05",
			DisableLevelTruncation: true,
		})
		log.SetReportCaller(true)

		HandleSafrp()
	})
}

func GetConn(ip, port string) net.Conn {
	c, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		panic(err)
	}
	return c
}

func HandleSafrp() {
	conn := make(chan string, 1024)
	go func() {
		c, err := net.Dial("tcp", conf.ServerIP + ":" + conf.ServerPort)
		if err != nil {
			panic(err)
		}
		buf := make([]byte, 1024)
		for {
			n, err := c.Read(buf)
			if err != nil {
				panic(err)
			}
			conn <- string(buf[:n])
		}
	}()
	go func() {
		for {
			select {
			case c := <-conn:
				go func(s string) {
					c1, err := net.Dial("tcp", conf.ServerIP + ":" + conf.ClientPort)
					if err != nil {
						panic(err)
					}
					c2 := GetConn(conf.HTTPIP, conf.HTTPPort)
					common.HandleConn(c1, c2)
				}(c)
			}
		}
	}()
}

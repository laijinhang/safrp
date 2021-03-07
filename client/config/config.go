package config

import (
	"gopkg.in/ini.v1"
	"safrp/client/log"
)

type Config struct {
	SafrpServerIP   string
	SafrpServerPort string
	ConnNum int
	Password   string
	HTTPIP     string
	HTTPPort   string
	Protocol   string
	PipeNum    int
}

func (this *Config)GetPipeNum() int {
	return this.PipeNum
}

func init() {
	loadConfig()
}

func GetConfig() *Config {
	return &conf
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
	log := log.GetLog()
	if err != nil {
		log.Fatal("Fail to read file: ", err)
	}
	serverIni := cfg.Section("server")
	conf.SafrpServerIP = serverIni.Key("ip").String()
	conf.SafrpServerPort = serverIni.Key("port").String()
	conf.Password = serverIni.Key("password").String()

	proxyIni := cfg.Section("proxy")
	conf.HTTPIP = proxyIni.Key("ip").String()
	conf.HTTPPort = proxyIni.Key("port").String()
	conf.Protocol = proxyIni.Key("protocol").String()

	conf.ConnNum = func(v int, e error) int {
		if e != nil {
			panic(e)
		}
		return v
	}(cfg.Section("proxy").Key("conn_num").Int())
	conf.PipeNum = func(v int, e error) int {
		if e != nil {
			panic(e)
		}
		return v
	}(cfg.Section("").Key("pipe_num").Int())

	log.Infoln("load safrp.ini ...")
	log.Infoln("server-ip:", conf.SafrpServerIP)
	log.Infoln("server-port:", conf.SafrpServerPort)
	log.Infoln("server-password:", conf.Password)
	log.Infoln("conn_num:", conf.ConnNum)
	log.Infoln("http-ip:", conf.HTTPIP)
	log.Infoln("http-port:", conf.HTTPPort)
	log.Infoln()
}
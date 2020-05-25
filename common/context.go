package common

type Context struct {
	Conf Config
	NumberPool *NumberPool
	ReadDate chan DataPackage
	SendData chan DataPackage
	DateLength int
	IP string
	Port string
	Proxy string
}
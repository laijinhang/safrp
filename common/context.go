package common

import "github.com/sirupsen/logrus"

type Context struct {
	Conf       interface{}
	NumberPool *NumberPool
	ReadDate   chan DataPackage
	SendData   chan DataPackage
	DateLength int
	IP         string
	Port       string
	Protocol   string
	Conn       interface{}
	Log 	  *logrus.Logger
}

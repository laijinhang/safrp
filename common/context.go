package common

import (
	"context"
	"github.com/sirupsen/logrus"
)

type Context struct {
	Conf       interface{}
	UnitId		int
	NumberPool *NumberPool
	ReadDate   chan DataPackage
	SendData   chan DataPackage
	DateLength int
	IP         string
	Port       string
	Protocol   string
	Conn       interface{}
	Log 	  *logrus.Logger
	Expand	   interface{}
	Ctx 	   context.Context
}

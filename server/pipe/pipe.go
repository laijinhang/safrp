package pipe

import (
	"github.com/sirupsen/logrus"
)

type pipe struct {
	Number int	// 编号
	log *logrus.Logger
}

func NewPipe(number int, log logrus.Logger) *pipe {
	return &pipe{
		Number: number,
		log:    &log,
	}
}

func (p *pipe)Run()  {

}
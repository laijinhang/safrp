package pipe

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
	"net"
	"safrp/common"
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
	p.log.Printf("启动pipe%d\n", p.Number+1)

	ctx := common.Context{
		Conf:       confs[number],
		UnitId:     p.Number + 1,
		NumberPool: common.NewNumberPool(uint64(confs[number].ExtranetConnNum), 1),
		SendData:   make(chan common.DataPackage, 1000),
		ReadDate:   make(chan common.DataPackage, 1000),
		Log:        log,
		Protocol:   common.GetL3Protocol(confs[number].Protocol),
	}
	ctx1 := ctx
	ctx2 := ctx

	sendChan := make([]chan common.DataPackage, ctx.Conf.(Config).PipeNum)
	readChan := make([]chan common.DataPackage, ctx.Conf.(Config).PipeNum)
	for i := 0; i < ctx.Conf.(Config).PipeNum; i++ {
		sendChan[i] = make(chan common.DataPackage)
		readChan[i] = make(chan common.DataPackage)
	}
	connChan := make([]chan common.DataPackage, confs[number].ExtranetConnNum)

	ctx1.Expand = Context{
		PipeName: logrus.Fields{
			"pipe": fmt.Sprint(number + 1),
		},
		ConnManage:         make([]net.Conn, ctx.Conf.(Config).PipeNum+1),
		PipeConnControllor: make(chan int, ctx.Conf.(Config).PipeNum),
		SafrpSendChan:      sendChan,
		SafrpReadChan:      readChan,
		ConnNumberPool:     common.NewNumberPool(uint64(ctx.Conf.(Config).PipeNum), uint64(1)),
		ConnDataChan:       connChan}
	ctx2.Expand = Context{
		PipeName: logrus.Fields{
			"pipe": fmt.Sprint(number + 1),
		},
		ConnManage:         make([]net.Conn, ctx.Conf.(Config).PipeNum+1),
		ConnClose:          make([]chan bool, ctx.Conf.(Config).PipeNum+1),
		SafrpSendChan:      sendChan,
		SafrpReadChan:      readChan,
		PipeConnControllor: make(chan int, ctx.Conf.(Config).PipeNum),
		ConnNumberPool:     common.NewNumberPool(uint64(ctx.Conf.(Config).PipeNum), uint64(1)),
		ConnDataChan:       connChan}

	es := common.UnitFactory(ctx.Conf.(Config).Protocol, ctx.Conf.(Config).IP, ctx.Conf.(Config).ExtranetPort)
	ss := common.UnitFactory(ctx.Conf.(Config).Protocol, ctx.Conf.(Config).IP, ctx.Conf.(Config).ServerPort)

	extranetServer.Register(&ctx1, &es)
	safrpServer.Register(&ctx2, &ss)

	go func() {
		// 对外
		ExtranetServer(&ctx1)
	}()
	func() {
		ctxExit, chanExit := context.WithCancel(context.Background())
		defer func() {
			chanExit() // 通知该服务下的所有协程退出
		}()
		ctx2.Ctx = ctxExit
		// 对safrp客户端
		SafrpServer(&ctx2)
	}()
}
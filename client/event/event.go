package event

// 事件
type Event interface {
	Publish()	// 发布事件
	Subscribe()	// 订阅事件
}

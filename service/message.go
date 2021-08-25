package service

type Event string

const (
	EventResize  Event = "resize"
	EventSendKey Event = "sendKey"
	EventClose   Event = "close"
)

type Message struct {
	Event Event
	Data  interface{}
}

package message

type MessageBus struct {
	Subscribers map[string][]chan<- Message
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		Subscribers: make(map[string][]chan<- Message),
	}
}

func (mb *MessageBus) Subscribe(messageType string, subscriber chan<- Message) {
	mb.Subscribers[messageType] = append(mb.Subscribers[messageType], subscriber)
}

func (mb *MessageBus) Publish(ms Message) {
	subscribers := mb.Subscribers[ms.Type]

	for _, subscriber := range subscribers {
		subscriber <- ms
	}
}

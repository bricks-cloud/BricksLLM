package message

import (
	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"go.uber.org/zap"
)

type Consumer struct {
	messageChan         <-chan Message
	done                chan bool
	log                 *zap.Logger
	numOfEventConsumers int
	handle              func(Message, map[string]string) error
	openAiUrls          map[string]string
}

type recorder interface {
	RecordKeySpend(keyId string, micros int64, costLimitUnit key.TimeUnit) error
	RecordUserSpend(userId string, micros int64, costLimitUnit key.TimeUnit) error
	RecordEvent(e *event.Event) error
}

func NewConsumer(mc <-chan Message, log *zap.Logger, num int, handle func(Message, map[string]string) error, openAiUrls map[string]string) *Consumer {
	return &Consumer{
		messageChan:         mc,
		done:                make(chan bool),
		log:                 log,
		numOfEventConsumers: num,
		handle:              handle,
		openAiUrls:          openAiUrls,
	}
}

func (c *Consumer) StartEventMessageConsumers() {
	for i := 0; i < c.numOfEventConsumers; i++ {
		go func() {
			for {
				select {
				case <-c.done:
					c.log.Info("event message consumer stoped...")
					return

				case m := <-c.messageChan:
					err := c.handle(m, c.openAiUrls)
					if err != nil {
						continue
					}

					continue
				}
			}
		}()
	}
}

func (c *Consumer) Stop() {
	c.log.Info("shutting down consumer...")

	c.done <- true
}

package messaging

import (
	"context"
	"github.com/halacs/haltonika/config"
)

type CustomerHandler func(data interface{}) error

type Messaging struct {
	ctx       context.Context
	consumers []CustomerHandler
}

func NewMessaging(ctx context.Context) *Messaging {
	return &Messaging{
		ctx: ctx,
	}
}

func (m *Messaging) Publish(data interface{}) {
	log := config.GetLogger(m.ctx)

	for _, customerFunc := range m.consumers {
		err := customerFunc(data)
		if err == nil {
			// ack
			log.Debugf("Data forwarded and processed.")
		} else {
			log.Errorf("Failed to forward data. %v", err)
		}
	}
}

func (m *Messaging) Subscribe(customerFunc CustomerHandler) {
	m.consumers = append(m.consumers, customerFunc)
}

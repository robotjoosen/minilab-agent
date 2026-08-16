package healthstats

import (
	"context"
	"log/slog"

	"github.com/robotjoosen/go-rabbit"
	"github.com/wagslane/go-rabbitmq"
)

// Subscribe consumes health.ping messages and updates store with each parsed message.
// It blocks until ctx is cancelled.
func Subscribe(conn *rabbitmq.Conn, exchange, routingKey, queueName string, store *Store, ctx context.Context) error {
	return rabbit.RunConsumer(conn, exchange, []string{routingKey}, queueName,
		func(d rabbitmq.Delivery) rabbitmq.Action {
			stats, err := ParseHealthMessage(d.Body)
			if err != nil {
				slog.Error("failed to parse health message", slog.String("error", err.Error()))
				return rabbitmq.NackDiscard
			}

			store.Update(stats)

			return rabbitmq.Ack
		},
		ctx,
	)
}

package healthstats

import (
	"context"
	"log/slog"

	"github.com/robotjoosen/go-rabbit"
	"github.com/wagslane/go-rabbitmq"
)

// Subscribe consumes health.ping messages and updates store with each one published by
// hostname. It blocks until ctx is cancelled.
func Subscribe(conn *rabbitmq.Conn, exchange, routingKey, queueName, hostname string, store *Store, ctx context.Context) error {
	return rabbit.RunConsumer(conn, exchange, []string{routingKey}, queueName,
		func(d rabbitmq.Delivery) rabbitmq.Action {
			return handleMessage(d.Body, hostname, store)
		},
		ctx,
	)
}

// handleMessage parses a health.ping payload and applies it to store only if it was
// published by hostname. go-health-service publishes every host's ping with the same
// literal routing key, so every agent's queue receives every host's messages -- most
// deliveries are expected to belong to another host and are acknowledged without being
// applied, rather than treated as an error.
func handleMessage(body []byte, hostname string, store *Store) rabbitmq.Action {
	host, stats, err := ParseHealthMessage(body)
	if err != nil {
		slog.Error("failed to parse health message", slog.String("error", err.Error()))
		return rabbitmq.NackDiscard
	}

	if host != hostname {
		return rabbitmq.Ack
	}

	store.Update(stats)

	return rabbitmq.Ack
}

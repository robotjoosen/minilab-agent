// minilab-agent/cmd/app/setup.go

package main

import (
	"log/slog"
	"os"

	"github.com/robotjoosen/minilab-agent/pkg/env"
)

const (
	defaultMode                 = "DEV"
	defaultLogLevel             = "INFO"
	defaultMessageBusURL        = "amqp://guest:guest@localhost:5672"
	defaultMessageBusExchange   = "health"
	defaultMessageBusRoutingKey = "health.ping"
	defaultHTTPListenAddr       = ":9100"
	defaultMDNSServiceName      = "_minilab-agent._tcp"
)

type Environment struct {
	Mode                 string     `mapstructure:"MODE"`
	LogLevel             slog.Level `mapstructure:"LOG_LEVEL"`
	MessagebusURL        string     `mapstructure:"MESSAGE_BUS_URL"`
	MessageBusExchange   string     `mapstructure:"MESSAGE_BUS_EXCHANGE"`
	MessageBusRoutingKey string     `mapstructure:"MESSAGE_BUS_ROUTINGKEY"`
	HTTPListenAddr       string     `mapstructure:"HTTP_LISTEN_ADDR"`
	MDNSServiceName      string     `mapstructure:"MDNS_SERVICE_NAME"`
}

func initLog(level slog.Level) {
	hostname, err := os.Hostname()
	if err != nil {
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})).
		With(slog.String("hostname", hostname)))
}

func loadEnv() Environment {
	e, err := env.Load[Environment](map[string]any{
		"MODE":                   defaultMode,
		"LOG_LEVEL":              defaultLogLevel,
		"MESSAGE_BUS_URL":        defaultMessageBusURL,
		"MESSAGE_BUS_EXCHANGE":   defaultMessageBusExchange,
		"MESSAGE_BUS_ROUTINGKEY": defaultMessageBusRoutingKey,
		"HTTP_LISTEN_ADDR":       defaultHTTPListenAddr,
		"MDNS_SERVICE_NAME":      defaultMDNSServiceName,
	})
	if err != nil {
		slog.Error("failed to load environment", "err", err.Error())
		os.Exit(1)
	}

	return e
}

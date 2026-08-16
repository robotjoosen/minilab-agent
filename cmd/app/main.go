// minilab-agent/cmd/app/main.go

package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/robotjoosen/go-rabbit"
	"github.com/robotjoosen/minilab-agent/internal/docker"
	"github.com/robotjoosen/minilab-agent/internal/exec"
	"github.com/robotjoosen/minilab-agent/pkg/discovery"
	"github.com/robotjoosen/minilab-agent/pkg/handler/capabilities"
	"github.com/robotjoosen/minilab-agent/pkg/handler/metrics"
	"github.com/robotjoosen/minilab-agent/pkg/healthstats"
	"github.com/robotjoosen/minilab-agent/pkg/mdnsadvertise"
	"github.com/robotjoosen/minilab-agent/pkg/server"
	"github.com/wagslane/go-rabbitmq"
)

const maxConnectRetries = 100

func main() {
	e := loadEnv()
	initLog(e.LogLevel)

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := &healthstats.Store{}

	// The message bus connection retries for up to ~200s if RabbitMQ is
	// unreachable. That must not block HTTP/mDNS from starting - a
	// monitoring agent should keep answering /capabilities and /metrics
	// even while the broker is down, so this runs in the background and
	// the rest of main() proceeds regardless of its outcome.
	go func() {
		conn, err := connectMessageBus(ctx, e.MessagebusURL)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				slog.Info("message bus connection aborted due to shutdown", slog.String("error", err.Error()))
				return
			}
			slog.Error("failed to connect to message bus", slog.String("error", err.Error()))
			return
		}

		if err := healthstats.Subscribe(conn, e.MessageBusExchange, e.MessageBusRoutingKey, "minilab-agent-"+hostname, store, ctx); err != nil {
			slog.Error("health subscriber stopped", slog.String("error", err.Error()))
		}
	}()

	dockerClient, err := docker.New()
	if err != nil {
		slog.Error("failed to create docker client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	aggregator := &discovery.Aggregator{
		Systemd: exec.New(),
		Docker:  dockerClient,
	}

	discoverer := discovery.NewCachingDiscoverer(aggregator, discovery.CacheTTL)

	capsHandler := &capabilities.Handler{
		Discoverer: discoverer,
		Hostname:   hostname,
	}

	metricsHandler := &metrics.Handler{
		Discoverer: discoverer,
		HostStats:  store,
	}

	_, portStr, err := net.SplitHostPort(e.HTTPListenAddr)
	if err != nil {
		panic(err)
	}

	portNum, err := strconv.Atoi(portStr)
	if err != nil {
		panic(err)
	}

	closer, err := mdnsadvertise.Start(e.MDNSServiceName, hostname, portNum)
	if err != nil {
		slog.Error("failed to start mDNS responder", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer closer.Close()

	srv := &server.Server{Port: portNum}
	srv.InitialiseRoutes(map[string]http.HandlerFunc{
		"GET /capabilities": capsHandler.Handle,
		"GET /metrics":      metricsHandler.Handle,
	})
	srv.Run()

	slog.Info("minilab-agent started", slog.String("http_addr", e.HTTPListenAddr))

	<-ctx.Done()

	srv.Stop()
}

// connectMessageBus waits for the message bus to become reachable and then
// connects to it, retrying every 2 seconds up to maxConnectRetries times. It
// is ctx-aware: a canceled ctx (e.g. shutdown signal) stops the retry loop
// promptly instead of waiting out the full retry budget.
func connectMessageBus(ctx context.Context, u string) (*rabbitmq.Conn, error) {
	mbu, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	retries := 0
	for {
		if retries >= maxConnectRetries {
			return nil, errors.New("cannot connect to message bus")
		}

		if _, err := net.DialTimeout("tcp", mbu.Host, 1*time.Second); err != nil {
			retries++
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(2 * time.Second):
			}
			continue
		}

		break
	}

	conn, err := rabbit.NewConnection(u)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

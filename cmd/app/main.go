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

	store := new(healthstats.Store)

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

	discoverer := discovery.NewCachingDiscoverer(discovery.DiscovererFunc(discovery.Discover), discovery.CacheTTL)

	capsHandler := &capabilities.Handler{
		Discoverer: discoverer,
		Hostname:   hostname,
	}

	metricsHandler := &metrics.Handler{
		Discoverer: discoverer,
		HostStats:  store,
	}

	portNum := mustParseHostPort(e.HTTPListenAddr)

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

func mustParseHostPort(host string) int {
	_, portStr, err := net.SplitHostPort(host)
	if err != nil {
		panic(err)
	}

	portNum, err := strconv.Atoi(portStr)
	if err != nil {
		panic(err)
	}

	return portNum
}

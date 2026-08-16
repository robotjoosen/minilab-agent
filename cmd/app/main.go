// minilab-agent/cmd/app/main.go
//go:build linux
// +build linux

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
	"github.com/robotjoosen/minilab-agent/pkg/healthstats"
	"github.com/robotjoosen/minilab-agent/pkg/httpapi"
	"github.com/robotjoosen/minilab-agent/pkg/mdnsadvertise"
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

	conn := connectMessageBus(e.MessagebusURL)
	go func() {
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

	server := &httpapi.Server{
		Discoverer: aggregator,
		HostStats:  store,
		Hostname:   hostname,
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

	httpServer := &http.Server{Addr: e.HTTPListenAddr, Handler: server.Routes()}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server stopped", slog.String("error", err.Error()))
		}
	}()

	slog.Info("minilab-agent started", slog.String("http_addr", e.HTTPListenAddr))

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func connectMessageBus(u string) *rabbitmq.Conn {
	mbu, err := url.Parse(u)
	if err != nil {
		panic(err)
	}

	retries := 0
	for {
		if retries >= maxConnectRetries {
			panic(errors.New("cannot connect to message bus"))
		}

		if _, err := net.DialTimeout("tcp", mbu.Host, 1*time.Second); err != nil {
			retries++
			<-time.NewTicker(2 * time.Second).C
			continue
		}

		break
	}

	conn, err := rabbit.NewConnection(u)
	if err != nil {
		panic(err)
	}

	return conn
}

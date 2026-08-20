package mdnsadvertise

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/hashicorp/mdns"
)

const (
	outboundIPMaxRetries = 100
	outboundIPRetryDelay = 2 * time.Second
)

// serverCloser adapts mdns.Server's Shutdown() method to io.Closer's Close() interface.
type serverCloser struct {
	server *mdns.Server
}

// Close shuts down the mDNS server.
func (sc *serverCloser) Close() error {
	return sc.server.Shutdown()
}

// Start begins responding to mDNS queries for serviceName until the returned io.Closer is
// closed (or the process exits). This is how the agent stays discoverable for as long as it
// runs, satisfying "advertises itself until shutdown" without a hand-rolled broadcast loop.
func Start(ctx context.Context, serviceName, instanceName string, port int) (io.Closer, error) {
	host, err := os.Hostname()
	if err != nil {
		host = instanceName
	}

	ip, err := outboundIP(ctx)
	if err != nil {
		return nil, fmt.Errorf("determining address to advertise: %w", err)
	}

	service, err := mdns.NewMDNSService(instanceName, serviceName, "", "", port, []net.IP{ip}, []string{host})
	if err != nil {
		return nil, err
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return nil, err
	}

	return &serverCloser{server: server}, nil
}

// outboundIP returns the address of the interface this host would use to reach the network.
//
// Left unspecified, mdns.NewMDNSService resolves the local hostname via net.LookupIP, which
// goes through /etc/hosts -- and Debian-family distros (Raspberry Pi OS included) commonly map
// the hostname to the 127.0.1.1 loopback alias there. That address would get advertised
// verbatim, making the agent unreachable from any other host on the LAN. Dialing UDP doesn't
// send any packets; it only asks the OS to resolve the route, which sidesteps /etc/hosts
// entirely and reliably lands on the real outbound interface.
//
// On boot, systemd can start this service before the network interface has a route --
// After=network.target only waits for networking services to start, not for an address to be
// configured. Retrying (mirroring connectMessageBus's approach) rides out that window instead
// of failing immediately.
func outboundIP(ctx context.Context) (net.IP, error) {
	var lastErr error
	for retries := 0; retries < outboundIPMaxRetries; retries++ {
		conn, err := net.Dial("udp", "8.8.8.8:80")
		if err == nil {
			defer conn.Close()

			addr, ok := conn.LocalAddr().(*net.UDPAddr)
			if !ok {
				return nil, fmt.Errorf("unexpected local address type %T", conn.LocalAddr())
			}
			return addr.IP, nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(outboundIPRetryDelay):
		}
	}
	return nil, lastErr
}

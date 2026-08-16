package mdnsadvertise

import (
	"io"
	"os"

	"github.com/hashicorp/mdns"
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
func Start(serviceName, instanceName string, port int) (io.Closer, error) {
	host, err := os.Hostname()
	if err != nil {
		host = instanceName
	}

	service, err := mdns.NewMDNSService(instanceName, serviceName, "", "", port, nil, []string{host})
	if err != nil {
		return nil, err
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return nil, err
	}

	return &serverCloser{server: server}, nil
}

// Command scan browses the LAN for minilab-agent instances over
// mDNS and prints what each one reports on /capabilities. It's a debug
// tool for a human to run from their own machine, not something deployed
// alongside the agent itself.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/mdns"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
)

type capabilitiesResponse struct {
	Device   string          `json:"device"`
	Services domain.Services `json:"services"`
}

func main() {
	service := flag.String("service", "_minilab-agent._tcp", "mDNS service name to scan for")
	timeout := flag.Duration("timeout", 3*time.Second, "how long to listen for mDNS responses")
	httpTimeout := flag.Duration("http-timeout", 15*time.Second, "how long to wait for each agent's /capabilities response")
	flag.Parse()

	entries, err := scan(*service, *timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mDNS query failed: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("no minilab-agent instances found")
		return
	}

	client := &http.Client{Timeout: *httpTimeout}
	for i, entry := range entries {
		if i > 0 {
			fmt.Println()
		}
		printEntry(client, entry)
	}
}

// scan browses for service on the local network for timeout, returning
// every instance that responded.
func scan(service string, timeout time.Duration) ([]*mdns.ServiceEntry, error) {
	entriesCh := make(chan *mdns.ServiceEntry, 16)
	var entries []*mdns.ServiceEntry
	done := make(chan struct{})
	go func() {
		for entry := range entriesCh {
			entries = append(entries, entry)
		}
		close(done)
	}()

	params := mdns.DefaultParams(service)
	params.Timeout = timeout
	params.Entries = entriesCh

	err := mdns.Query(params)
	close(entriesCh)
	<-done

	return entries, err
}

// printEntry queries entry's /capabilities endpoint and prints the result,
// or a single failure line if it couldn't be reached -- one unreachable
// agent shouldn't stop the rest of the scan from being reported.
func printEntry(client *http.Client, entry *mdns.ServiceEntry) {
	addr := entry.AddrV4
	if addr == nil {
		addr = entry.AddrV6
	}
	if addr == nil {
		fmt.Printf("%s  (no usable address)\n", entry.Host)
		return
	}
	hostPort := net.JoinHostPort(addr.String(), strconv.Itoa(entry.Port))

	fmt.Printf("%s  %s\n", entry.Host, hostPort)

	resp, err := client.Get(fmt.Sprintf("http://%s/capabilities", hostPort))
	if err != nil {
		fmt.Printf("  ! failed to reach /capabilities: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("  ! /capabilities returned %s\n", resp.Status)
		return
	}

	var caps capabilitiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&caps); err != nil {
		fmt.Printf("  ! failed to decode /capabilities response: %v\n", err)
		return
	}

	if len(caps.Services) == 0 {
		fmt.Println("  (no services discovered)")
		return
	}

	for _, svc := range caps.Services {
		fmt.Printf("  %-30s %-8s %-8s %s\n", svc.Name, svc.Type, svc.State.String(), svc.Version)
	}
}

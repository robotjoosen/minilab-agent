package mdnsadvertise

import (
	"context"
	"testing"
)

func TestOutboundIPIsNotLoopback(t *testing.T) {
	ip, err := outboundIP(context.Background())
	if err != nil {
		t.Fatalf("outboundIP() error = %v", err)
	}

	if ip.IsLoopback() {
		t.Fatalf("outboundIP() = %v, want a non-loopback address", ip)
	}
}

package mdnsadvertise

import "testing"

func TestOutboundIPIsNotLoopback(t *testing.T) {
	ip, err := outboundIP()
	if err != nil {
		t.Fatalf("outboundIP() error = %v", err)
	}

	if ip.IsLoopback() {
		t.Fatalf("outboundIP() = %v, want a non-loopback address", ip)
	}
}

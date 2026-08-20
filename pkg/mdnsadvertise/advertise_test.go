package mdnsadvertise_test

import (
	"context"
	"testing"

	"github.com/robotjoosen/minilab-agent/pkg/mdnsadvertise"
)

func TestStartAndCloseDoesNotError(t *testing.T) {
	closer, err := mdnsadvertise.Start(context.Background(), "_minilab-agent._tcp", "rocket", 9100)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer closer.Close()

	if closer == nil {
		t.Fatal("expected a non-nil closer")
	}
}

package discovery_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/robotjoosen/minilab-agent/pkg/discovery"
)

type fakeRunner struct {
	outputs map[string]string
}

func (f fakeRunner) Run(name string, args ...string) (string, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	out, ok := f.outputs[key]
	if !ok {
		return "", fmt.Errorf("no fake output for command: %s", key)
	}
	return out, nil
}

func TestDiscoverSystemdUnits(t *testing.T) {
	runner := fakeRunner{outputs: map[string]string{
		"systemctl list-units --type=service --all --no-legend --plain": "" +
			"nodered.service loaded active running Node-RED\n" +
			"mifare.service  loaded active exited  Mifare reader\n",
		"systemctl show nodered.service --property=ExecStart --value": "{ path=/opt/minilab/bin/nodered ; argv[]=/opt/minilab/bin/nodered ; ignore_errors=no ; start_time=... }",
		"systemctl show mifare.service --property=ExecStart --value":  "{ path=/opt/minilab/bin/mifare ; argv[]=/opt/minilab/bin/mifare ; ignore_errors=no ; start_time=... }",
	}}

	units, err := discovery.DiscoverSystemdUnits(runner)
	if err != nil {
		t.Fatalf("DiscoverSystemdUnits() error = %v", err)
	}

	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d: %+v", len(units), units)
	}

	if units[0].Name != "nodered.service" || !units[0].Active || units[0].ExecStart != "/opt/minilab/bin/nodered" {
		t.Fatalf("unexpected unit[0]: %+v", units[0])
	}

	if units[1].Name != "mifare.service" || !units[1].Active {
		t.Fatalf("unexpected unit[1]: %+v", units[1])
	}
}

func TestVersionFromBinaryUsesModTime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "somebinary")
	if err := os.WriteFile(path, []byte("fake"), 0o755); err != nil {
		t.Fatalf("failed to write fixture binary: %v", err)
	}

	mtime := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("failed to set mtime: %v", err)
	}

	got, err := discovery.VersionFromBinary(path)
	if err != nil {
		t.Fatalf("VersionFromBinary() error = %v", err)
	}

	want := mtime.Format(time.RFC3339)
	if got != want {
		t.Fatalf("VersionFromBinary() = %q, want %q", got, want)
	}
}

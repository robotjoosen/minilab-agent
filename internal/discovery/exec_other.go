//go:build !linux

package discovery

import "fmt"

func execCommand(name string, args ...string) (string, error) {
	return "", fmt.Errorf("systemd discovery only supported on Linux")
}

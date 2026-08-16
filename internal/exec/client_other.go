//go:build !linux

package exec

import "fmt"

func execCommand(name string, args ...string) (string, error) {
	return "", fmt.Errorf("systemd discovery only supported on Linux")
}

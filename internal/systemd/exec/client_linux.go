//go:build linux

package exec

import osexec "os/exec"

func execCommand(name string, args ...string) (string, error) {
	out, err := osexec.Command(name, args...).Output()

	return string(out), err
}

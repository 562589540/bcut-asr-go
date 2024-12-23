//go:build !windows
// +build !windows

package utils

import "os/exec"

func RunCommand(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	return cmd
}

//go:build !windows

package runtime

import (
	"os"
	"syscall"
)

// terminateProcess asks a child to shut down cleanly.
func terminateProcess(p *os.Process) error {
	return p.Signal(syscall.SIGTERM)
}

// sysProcAttr puts the child in its own process group on Linux so a stray
// signal aimed at the app does not reach cloudflared unexpectedly.
func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

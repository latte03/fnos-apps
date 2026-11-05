//go:build windows

package runtime

import (
	"os"
	"syscall"
)

// terminateProcess has no SIGTERM on Windows; kill directly so the developer
// build still behaves predictably. Production runs on Linux.
func terminateProcess(p *os.Process) error {
	return p.Kill()
}

// sysProcAttr is a no-op on Windows.
func sysProcAttr() *syscall.SysProcAttr {
	return nil
}

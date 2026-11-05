package runtime

import (
	"context"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// Spec is one child launch request. The tunnel token must travel in Env, never
// in Args, so it stays out of `ps`.
type Spec struct {
	Path string
	Args []string
	Env  []string
}

// Child is a running cloudflared process.
type Child interface {
	// Output streams combined stdout and stderr.
	Output() io.Reader
	// Wait blocks until the process exits.
	Wait() error
	// Terminate asks the process to stop (SIGTERM on Linux).
	Terminate() error
	// Kill ends the process unconditionally.
	Kill() error
	// Close releases the output pipe.
	Close() error
	// Pid returns the process id, 0 when unknown.
	Pid() int
}

// Factory starts child processes. Tests and demo mode inject their own.
type Factory interface {
	Start(ctx context.Context, spec Spec) (Child, error)
}

// Prober answers readiness and metrics for one running attempt.
type Prober interface {
	Ready(ctx context.Context, startedAt time.Time) bool
	Metrics(ctx context.Context, startedAt time.Time) (model.RuntimeMetrics, bool)
}

// ExecFactory is the production factory.
type ExecFactory struct{}

// Start launches spec.Path and returns a child piping both streams.
func (ExecFactory) Start(_ context.Context, spec Spec) (Child, error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(spec.Path, spec.Args...)
	cmd.Env = spec.Env
	cmd.Stdout = pw
	cmd.Stderr = pw
	cmd.SysProcAttr = sysProcAttr()
	if err := cmd.Start(); err != nil {
		_ = pr.Close()
		_ = pw.Close()
		return nil, err
	}
	// The child owns the write end from here on; closing our copy makes the
	// reader see EOF as soon as the child exits.
	_ = pw.Close()
	return &execChild{cmd: cmd, out: pr}, nil
}

type execChild struct {
	cmd *exec.Cmd
	out *os.File
}

func (c *execChild) Output() io.Reader { return c.out }

func (c *execChild) Wait() error { return c.cmd.Wait() }

func (c *execChild) Terminate() error {
	if c.cmd.Process == nil {
		return nil
	}
	return terminateProcess(c.cmd.Process)
}

func (c *execChild) Kill() error {
	if c.cmd.Process == nil {
		return nil
	}
	return c.cmd.Process.Kill()
}

func (c *execChild) Close() error { return c.out.Close() }

func (c *execChild) Pid() int {
	if c.cmd.Process == nil {
		return 0
	}
	return c.cmd.Process.Pid
}

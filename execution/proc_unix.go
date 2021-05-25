// +build !windows

package execution

import (
	"context"
	"os/exec"
	"syscall"
)

// on unix based systems, this struct is a no-op struct as job objects are irrelevant for these types of systems
type jobObjectHandler struct {}

// create a no-op job object handler
func newJobObjHandler() (*jobObjectHandler, error) {
	return &jobObjectHandler{}, nil
}

// no-op
func (jh *jobObjectHandler) assignCmdToJobObj(_ *exec.Cmd) error {
	return nil
}

// no-op
func (jh *jobObjectHandler) killJobObj() error {
	return nil
}

func getCommand(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, "sh", "-c", command)
}

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcess(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

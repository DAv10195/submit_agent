// +build windows

package execution

import (
	"context"
	"fmt"
	"golang.org/x/sys/windows"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// we use this struct to retrieve process handle (which is unexported)
// from os.Process using unsafe operation
type process struct {
	Pid    int
	Handle uintptr
}

// in windows, we'll create a job object (https://docs.microsoft.com/en-us/windows/win32/procthread/job-objects)
// to kill a command and children processes created by it (actually the entire hierarchy created by the root process)
type jobObjectHandler struct {
	mutex	*sync.Mutex
	jobObj	windows.Handle
	killed	bool
}

// create the job object
func newJobObjHandler() (*jobObjectHandler, error) {
	jh := &jobObjectHandler{mutex: &sync.Mutex{}, killed: false}
	jobObjCreationErrFormat := "error creating job object: %v"
	jobObj, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf(jobObjCreationErrFormat, err)
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(jobObj, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		return nil, fmt.Errorf(jobObjCreationErrFormat, err)
	}
	jh.jobObj = jobObj
	return jh, nil
}

// assign the given command to the job object managed by the handler
func (jh *jobObjectHandler) assignCmdToJobObj(cmd *exec.Cmd) error {
	jh.mutex.Lock()
	defer jh.mutex.Unlock()
	if err := windows.AssignProcessToJobObject(jh.jobObj, windows.Handle((*process)(unsafe.Pointer(cmd.Process)).Handle)); err != nil {
		return fmt.Errorf("error assigning cmd processes to job object: %v", err)
	}
	return nil
}

// kill the job object and the processes assigned to it by closing the corresponding windows handler
func (jh *jobObjectHandler) killJobObj() error {
	jh.mutex.Lock()
	defer jh.mutex.Unlock()
	defer func() { jh.killed = true }()
	if !jh.killed {
		if err := windows.CloseHandle(jh.jobObj); err != nil {
			return fmt.Errorf("error killing job object processes: %v", err)
		}
	}
	return nil
}

func getCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: strings.Join([]string{"/c", command}, " "),
	}
	return cmd
}

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP
}

func killProcess(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}

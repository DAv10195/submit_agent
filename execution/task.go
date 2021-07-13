package execution

import (
	"bytes"
	"context"
	"fmt"
	commons "github.com/DAv10195/submit_commons"
	"github.com/DAv10195/submit_commons/containers"
	"github.com/DAv10195/submit_commons/encryption"
	"github.com/DAv10195/submit_commons/fsclient"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type TaskExecution struct {
	Command			string
	Timeout			int
	Dependencies	*containers.StringSet
	Encryption		encryption.Encryption
	FsHost			string
	FsPort			int
	FsUser			string
	FsPassword		string
}

func (e *TaskExecution) Execute(testRun bool) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := getCommand(ctx, e.Command)
	workDir := fmt.Sprintf("submit_agent_exec_%s", commons.GenerateUniqueId())
	workingDir := filepath.Join(os.TempDir(), workDir)
	if err := os.MkdirAll(workingDir, 0700); err != nil {
		logger.WithError(err).Errorf("error creating task execution working directory [ %s ]", workingDir)
		return "", err
	}
	defer func() {
		if err := os.RemoveAll(workingDir); err != nil {
			logger.WithError(err).Errorf("error removing task execution working directory [ %s ]", workingDir)
		}
	}()
	if !testRun {
		fsc, err := fsclient.NewFileServerClient(fmt.Sprintf("http://%s:%d", e.FsHost, e.FsPort), e.FsUser, e.FsPassword, logger, e.Encryption)
		if err != nil {
			return "", err
		}
		for _, fsPath := range e.Dependencies.Slice() {
			f, err := os.Create(filepath.Join(workingDir, filepath.Base(fsPath)))
			if err != nil {
				return "", err
			}
			if _, err := fsc.DownloadFile(fsPath, f); err != nil {
				return "", err
			}
		}
	}
	cmd.Dir = workingDir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	setProcessGroup(cmd)
	// no-op which always return a nil error on unix, but on windows it returns a job object handler struct.
	// On windows, we must create the job object and register the cmd to it or otherwise sub processes spawned by the
	// command will remain as zombies
	jobObjHandler, jobObjCreationErr := newJobObjHandler()
	if jobObjCreationErr != nil {
		return "", jobObjCreationErr
	}
	// even if no timeout defined, make sure we kill the job object (only on windows, as this is a no-op for unix)
	defer func() {
		if err := jobObjHandler.killJobObj(); err != nil {
			logger.WithError(err).Error("error killing job object")
		}
	}()
	var timer *time.Timer
	if e.Timeout > 0 {
		timer = time.AfterFunc(time.Duration(e.Timeout) * time.Second, func() {
			cancel()
			errKillProc := killProcess(cmd)
			// no-op on unix, but on windows this kills the job object which the all of the processes of the command are
			// assigned to (thus killing the entire process hierarchy)
			errKillJobObj := jobObjHandler.killJobObj()
			if errKillProc != nil  || errKillJobObj != nil {
				err := fmt.Errorf("process killing error: [ %v ] ; job object killing error: [ %v ]", errKillProc, errKillJobObj)
				logger.WithError(err).Errorf("error killing command with process pid: [ %d ]", cmd.Process.Pid)
			} else {
				logger.Debugf("terminated/killed process [ %d ] successfully", cmd.Process.Pid)
			}
		})
		defer timer.Stop()
	} else {
		return "", fmt.Errorf("non positive integer given as timeout value (%d)", e.Timeout)
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	// no-op on unix, but adds the cmd to the job object on windows
	if err := jobObjHandler.assignCmdToJobObj(cmd); err != nil {
		logger.WithError(err).Error("error assigning command to job object")
	}
	err := cmd.Wait()
	if err != nil {
		return "", err
	}
	if timer != nil {
		timer.Stop()
	}
	cmdOutput := output.String()
	if ctx.Err() == context.Canceled {
		return "", fmt.Errorf("task execution timed out")
	} else if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// best effort to determine the exit status. This should work on linux, darwin and windows
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				if cmdOutput != "" {
					return "", fmt.Errorf("error: [ %v ], exit status: [ %d ], output: [ %s ]", err, status.ExitStatus(), cmdOutput)
				}
				return "", fmt.Errorf("error: [ %v ], exit status: [ %d ]", err, status.ExitStatus())
			}
		}
		if cmdOutput != "" {
			return "", fmt.Errorf("error: [ %v ], output: [ %s ]", err, cmdOutput)
		}
		return "", fmt.Errorf("error: [ %v ]", err)
	}
	return cmdOutput, nil
}

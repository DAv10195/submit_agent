package agent

import (
	"encoding/json"
	"fmt"
	"github.com/DAv10195/submit_agent/execution"
	submitws "github.com/DAv10195/submit_commons/websocket"
	"sync"
	"sync/atomic"
	"time"
)

// handle keepalive response by advancing the read deadline on the connection to the submit server by one minute
func (a *Agent) handleKeepaliveResponse(_ []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	newReadDeadLine := time.Now().Add(time.Minute)
	if err := a.endpoint.conn.SetReadDeadline(newReadDeadLine); err != nil {
		logger.WithError(err).Errorf("error advancing read deadline to %v on connection to %s", newReadDeadLine.Format(time.RFC3339), a.endpoint.url)
	} else {
		logger.Debugf("keepalive response received and read deadline advanced to %s on connection to %s", newReadDeadLine.Format(time.RFC3339), a.endpoint.url)
	}
}

func (a *Agent) handleTask(payload []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	task := &submitws.Task{}
	if err := json.Unmarshal(payload, task); err != nil {
		logger.WithError(err).Errorf("error parsing task payload: %s", string(payload))
		return
	}
	if a.maxTasksSemaphore != nil {
		a.maxTasksSemaphore <- struct{}{}
		defer func() {
			<- a.maxTasksSemaphore
		}()
	}
	atomic.AddInt64(&a.numRunningTasks, 1)
	defer atomic.AddInt64(&a.numRunningTasks, -1)
	tr := &submitws.TaskResponse{Handler: task.ResponseHandler, Task: task.ID}
	te := &execution.TaskExecution{Command: task.Command, Timeout: task.Timeout, Dependencies: task.Dependencies, FsHost: a.config.SubmitFsHost, FsPort: a.config.SubmitFsPort, FsUser: a.config.SubmitFsUser, FsPassword: a.config.SubmitFsPassword, Encryption: a.encryption}
	logger.Infof("executing task: %s", string(payload))
	output, err := te.Execute(false)
	if err != nil {
		logger.WithError(err).Errorf("error executing task with id == %s", task.ID)
		tr.Status = submitws.TaskRespExecStatusErr
		tr.Payload = fmt.Sprintf("error executing task: %v", err)
	} else {
		logger.Infof("successfully executed task with id == %s", task.ID)
		tr.Status = submitws.TaskRespExecStatusOk
		tr.Payload = output
	}
	logger.Infof("queueing task with id == %s for sending", task.ID)
	if _, err := a.messageQueue.EnqueueObjectAsJSON(tr); err != nil {
		logger.WithError(err).Errorf("error queueing task response for sending (task id == %s)", tr.Task)
	}
}

func (a *Agent) handlers() map[string]serverMessageHandler {
	handlers := make(map[string]serverMessageHandler)
	handlers[submitws.MessageTypeKeepaliveResponse] = a.handleKeepaliveResponse
	handlers[submitws.MessageTypeTask] = a.handleTask
	return handlers
}

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	commons "github.com/DAv10195/submit_commons"
	"github.com/DAv10195/submit_commons/encryption"
	submitws "github.com/DAv10195/submit_commons/websocket"
	"github.com/beeker1121/goque"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// agent
type Agent struct {
	id					string
	encryption  		encryption.Encryption
	config				*Config
	endpoint 			*serverEndpoint
	numRunningTasks		int64
	maxTasksSemaphore	chan struct{}
	messageQueue		*goque.Queue
}

// create a new agent
func NewAgent(cfg *Config) (*Agent, error) {
	// initialize encryption
	keyFilePath := filepath.Join(cfg.CacheDir, encryptionKeyFileName)
	if err := encryption.GenerateAesKeyFile(keyFilePath); err != nil {
		return nil, fmt.Errorf("error initalizing encryption key file: %v", err)
	}
	msgQ, err := goque.OpenQueue(filepath.Join(cfg.CacheDir, queue))
	if err != nil {
		return nil, fmt.Errorf("error intializing message queue: %v", err)
	}
	a := &Agent{config: cfg, encryption: &encryption.AesEncryption{KeyFilePath: keyFilePath}, messageQueue: msgQ}
	if a.config.MaxRunningTasks > 0 {
		a.maxTasksSemaphore = make(chan struct{}, a.config.MaxRunningTasks)
	}
	if err := a.handleConfigEncryption(); err != nil {
		return nil, fmt.Errorf("error handling config encryption: %v", err)
	}
	// get agent id
	if err := a.resolveAgentId(filepath.Join(cfg.CacheDir, agentIdFileName)); err != nil {
		return nil, fmt.Errorf("error resolving agent id: %v", err)
	}
	return a, nil
}

// encrypt passwords in memory and in the config file, if required
func (a *Agent) handleConfigEncryption() error {
	var writeToConfRequired bool
	var err error
	if !strings.HasPrefix(a.config.SubmitServerPassword, encryptedPrefix) {
		writeToConfRequired = true
		a.config.SubmitServerPassword, err = a.encryption.Encrypt(a.config.SubmitServerPassword)
		if err != nil {
			return err
		}
	} else {
		a.config.SubmitServerPassword = strings.TrimPrefix(a.config.SubmitServerPassword, encryptedPrefix)
	}
	if !strings.HasPrefix(a.config.SubmitFsPassword, encryptedPrefix) {
		writeToConfRequired = true
		a.config.SubmitFsPassword, err = a.encryption.Encrypt(a.config.SubmitFsPassword)
		if err != nil {
			return err
		}
	} else {
		a.config.SubmitFsPassword = strings.TrimPrefix(a.config.SubmitFsPassword, encryptedPrefix)
	}
	if writeToConfRequired {
		if _, err = os.Stat(a.config.ConfFile); err != nil {
			if os.IsNotExist(err) {
				return nil
			} else {
				return err
			}
		}
		confLines, err := a.readConfLines()
		if err != nil {
			return err
		}
		for i := 0; i < len(confLines); i++ {
			if strings.Contains(confLines[i], flagServerPassword) {
				confLines[i] = fmt.Sprintf("%s: %s%s", flagServerPassword, encryptedPrefix, a.config.SubmitServerPassword)
			} else if strings.Contains(confLines[i], flagFileServerPassword) {
				confLines[i] = fmt.Sprintf("%s: %s%s", flagFileServerPassword, encryptedPrefix, a.config.SubmitFsPassword)
			}
		}
		if err := a.writeConfLines(confLines); err != nil {
			return err
		}
	}
	return nil
}

func (a *Agent) readConfLines() ([]string, error) {
	confFile, err := os.Open(a.config.ConfFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := confFile.Close(); err != nil {
			logger.WithError(err).Error("error closing config file after reading")
		}
	}()
	var confLines []string
	confScanner := bufio.NewScanner(confFile)
	for confScanner.Scan() {
		confLines = append(confLines, confScanner.Text())
	}
	return confLines, confScanner.Err()
}

func (a *Agent) writeConfLines(confLines []string) error {
	confFile, err := os.Create(a.config.ConfFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := confFile.Close(); err != nil {
			logger.WithError(err).Error("error closing config file after writing")
		}
	}()
	confWriter := bufio.NewWriter(confFile)
	for _, line := range confLines {
		if _, err := fmt.Fprintln(confWriter, line); err != nil {
			return err
		}
	}
	return confWriter.Flush()
}

func (a *Agent) resolveAgentId(idFilePath string) error {
	if _, err := os.Stat(idFilePath); os.IsNotExist(err) {
		a.id = commons.GenerateUniqueId()
		if err := ioutil.WriteFile(idFilePath, []byte(a.id), agentIdFilePerms); err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	}
	rawId, err := ioutil.ReadFile(idFilePath)
	if err != nil {
		return err
	}
	a.id = string(rawId)
	if len(a.id) != commons.UniqueIdLen {
		return fmt.Errorf("length of agent id file is different then the expected length [ %d ]", commons.UniqueIdLen)
	}
	return nil
}

// send a udp packet to the Google DNS servers in order to get the preferred outbound ip of the host networking
func (a *Agent) getOutboundIp() (string, error) {
	conn, err := net.Dial(udp, googleDnsServer)
	if err != nil {
		return "", err
	}
	outboundIp := conn.LocalAddr().(*net.UDPAddr).IP
	if err := conn.Close(); err != nil {
		return "", err
	}
	return outboundIp.To4().String(), nil
}

// create a new keepalive message
func (a *Agent) newKeepalive() (*submitws.Message, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	ip, err := a.getOutboundIp()
	if err != nil {
		return nil, err
	}
	keepalive := &submitws.Keepalive{
		OsType:    runtime.GOOS,
		IpAddress: ip,
		Hostname:  hostname,
		Architecture: runtime.GOARCH,
		NumRunningTasks: int(atomic.LoadInt64(&a.numRunningTasks)),
	}
	keepalivePayload, err := json.Marshal(keepalive)
	if err != nil {
		return nil, err
	}
	msg, err := submitws.NewMessage(submitws.MessageTypeKeepalive, keepalivePayload)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// send keepalive messages to the submit server
func (a *Agent) keepaliveLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	logger.Info("starting keepalive loop")
	keepalive, err := a.newKeepalive()
	if err != nil {
		logger.WithError(err).Error("error creating keepalive message. stopping keepalive loop")
		return
	}
	a.endpoint.write(keepalive)
	ticker := time.NewTicker(keepaliveInterval)
	defer ticker.Stop()
	for {
		select {
			case <- ticker.C:
				keepalive, err := a.newKeepalive()
				if err != nil {
					logger.WithError(err).Error("error creating keepalive message. stopping keepalive loop")
					return
				}
				a.endpoint.write(keepalive)
			case <- ctx.Done():
				logger.Info("stopping keepalive loop")
				return
		}
	}
}

// create a new message containing task responses
func (a *Agent) newMessageFromQueue() (*submitws.Message, error) {
	var responses []*submitws.TaskResponse
	for i := 0; i < maxMsgBatchFromQueue && a.messageQueue.Length() > 0; i++ { // send at most a single segment size
		taskRespItem, err := a.messageQueue.Dequeue()
		if err != nil {
			return nil, err
		}
		taskResp := &submitws.TaskResponse{}
		if err := taskRespItem.ToObjectFromJSON(taskResp); err != nil {
			return nil, err
		}
		responses = append(responses, taskResp)
	}
	if len(responses) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(&submitws.TaskResponses{Responses: responses})
	if err != nil {
		return nil, err
	}
	msg, err := submitws.NewMessage(submitws.MessageTypeTaskResponses, payload)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// send queued messages to the submit server
func (a *Agent) sendLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	logger.Info("starting send loop")
	qMsg, err := a.newMessageFromQueue()
	if err != nil {
		logger.WithError(err).Error("error creating queued message. stopping send loop")
		return
	}
	if qMsg != nil {
		logger.Info("send loop: sending queued messages")
		a.endpoint.write(qMsg)
	} else {
		logger.Debug("send loop: no queued messages to send")
	}
	ticker := time.NewTicker(sendQueueInterval)
	defer ticker.Stop()
	for {
		select {
		case <- ticker.C:
			qMsg, err := a.newMessageFromQueue()
			if err != nil {
				logger.WithError(err).Error("error creating queued message. stopping send loop")
				return
			}
			if qMsg != nil {
				logger.Info("send loop: sending queued messages")
				a.endpoint.write(qMsg)
			} else {
				logger.Debug("send loop: no queued messages to send")
			}
		case <- ctx.Done():
			logger.Info("stopping send loop")
			return
		}
	}
}

// run the agent
func (a *Agent) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if err := a.messageQueue.Close(); err != nil {
			logger.WithError(err).Error("error closing message queue")
		}
	}()
	// TODO: add option for TLS
	url := fmt.Sprintf("ws://%s:%d/%s/endpoint", a.config.SubmitServerHost, a.config.SubmitServerPort, submitws.Agents)
	a.endpoint = &serverEndpoint{
		id: a.id,
		url: url,
		mutex: &sync.RWMutex{},
		user: a.config.SubmitServerUser,
		password: a.config.SubmitServerPassword,
		encryption: a.encryption,
		handlers: a.handlers(),
	}
	agentWg := &sync.WaitGroup{}
	defer agentWg.Wait()
	var keepaliveCtx context.Context
	var keepaliveCtxCancel context.CancelFunc
	defer func() {
		if keepaliveCtxCancel != nil {
			keepaliveCtxCancel()
		}
	}()
	var sendCtx context.Context
	var sendCtxCancel context.CancelFunc
	defer func() {
		if sendCtxCancel != nil {
			sendCtxCancel()
		}
	}()
	// try initializing the connection loop each second
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
			case <- ticker.C:
				if !a.endpoint.isConnected() {
					if keepaliveCtxCancel != nil {
						keepaliveCtxCancel()
						keepaliveCtxCancel = nil
					}
					if sendCtxCancel != nil {
						sendCtxCancel()
						sendCtxCancel = nil
					}
					a.endpoint.connect(connInterval, ctx)
					if a.endpoint.isConnected() {
						// once the above call to connect is done, it means that the agent is now connected to the submit
						// server (in case the given context wasn't done...), so let's start 3 goroutines - one for reading
						// one for sending keepalive messages and one for sending queued messages
						keepaliveCtx, keepaliveCtxCancel = context.WithCancel(ctx)
						sendCtx, sendCtxCancel = context.WithCancel(ctx)
						agentWg.Add(3)
						go a.keepaliveLoop(keepaliveCtx, agentWg)
						go a.sendLoop(sendCtx, agentWg)
						go a.endpoint.readLoop(agentWg)
					}
				}
			case <- ctx.Done():
				a.endpoint.close()
				return
		}
	}
}

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	commons "github.com/DAv10195/submit_commons"
	"github.com/DAv10195/submit_commons/encryption"
	submitws "github.com/DAv10195/submit_commons/websocket"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// agent
type Agent struct {
	id			string
	encryption  encryption.Encryption
	config		*Config
	endpoint 	*serverEndpoint
}

// create a new agent
func NewAgent(cfg *Config) (*Agent, error) {
	// initialize encryption
	keyFilePath := filepath.Join(cfg.CacheDir, encryptionKeyFileName)
	if err := encryption.GenerateAesKeyFile(keyFilePath); err != nil {
		return nil, fmt.Errorf("error initalizing encryption key file: %v", err)
	}
	a := &Agent{config: cfg, encryption: &encryption.AesEncryption{KeyFilePath: keyFilePath}}
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
	confFile, err := os.Open(a.config.ConfFile)
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
		// TODO: fill real number of running tasks
		NumRunningTasks: 0,
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

// run the agent
func (a *Agent) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	// TODO: add option for TLS
	url := fmt.Sprintf("ws://%s:%d/%s", a.config.SubmitServerHost, a.config.SubmitServerPort, submitws.Agents)
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
					a.endpoint.connect(connInterval)
					// once the above call to connect is done, it means that the agent is now connected to the submit
					// server, so let's start 2 goroutines - one for reading and for sending keepalive messages
					keepaliveCtx, keepaliveCtxCancel = context.WithCancel(ctx)
					agentWg.Add(2)
					go a.keepaliveLoop(keepaliveCtx, agentWg)
					go a.endpoint.readLoop(agentWg)
				}
			case <- ctx.Done():
				a.endpoint.close()
				return
		}
	}
}

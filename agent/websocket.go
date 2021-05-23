package agent

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/DAv10195/submit_commons/encryption"
	submitws "github.com/DAv10195/submit_commons/websocket"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"time"
)

type serverMessageHandler func([]byte, *sync.WaitGroup)

// websocket communications endpoint to the submit server
type serverEndpoint struct {
	id         string
	url        string
	conn       *websocket.Conn
	mutex      *sync.RWMutex
	user       string
	password   string
	encryption encryption.Encryption
	connected  bool
	handlers   map[string]serverMessageHandler
}

// connect to the submit server. This method should be called only from the connect method as it is unsynchronized
func (e *serverEndpoint) _connect() error {
	decryptedPassword, err := e.encryption.Decrypt(e.password)
	if err != nil {
		return err
	}
	header := http.Header{}
	header.Set(authorization, fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", e.user, decryptedPassword)))))
	header.Set(submitws.AgentIdHeader, e.id)
	dialer := websocket.Dialer{HandshakeTimeout: connTimeout}
	var resp *http.Response
	e.conn, resp, err = dialer.Dial(e.url, header)
	if err != nil {
		if resp != nil {
			if err == websocket.ErrBadHandshake {
				return fmt.Errorf("handshake failed with status code [ %d ], err: %v", resp.StatusCode, err)
			}
			return fmt.Errorf("connection failed with status code [ %d ], err: %v", resp.StatusCode, err)
		}
		return err
	}
	return nil
}

// connect to the submit server. The given interval will be the time waited between connection attempts
func (e *serverEndpoint) connect(interval time.Duration, ctx context.Context) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	logger.Infof("connecting to %s...", e.url)
	if !e.connected {
		for {
			if err := e._connect(); err == nil {
				logger.Infof("successfully connected to %s", e.url)
				e.connected = true
				return
			} else {
				logger.WithError(err).Errorf("failed connecting to %s", e.url)
				if ctx.Err() != nil {
					break
				}
				time.Sleep(interval)
			}
		}
	}
}

// close the connection to the submit server
func (e *serverEndpoint) close() {
	e.mutex.Lock()
	defer func() {
		_ = recover()
		e.mutex.Unlock()
	}()
	if !e.connected {
		return
	}
	if err := e.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "bye bye")); err != nil {
		logger.WithError(err).Errorf("error sending closing message to %s", e.url)
	}
	e.connected = false
}

// send a message to the submit server
func (e *serverEndpoint) write(msg *submitws.Message) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if !e.connected {
		return
	}
	if err := e.conn.WriteMessage(websocket.BinaryMessage, msg.ToBinary()); err != nil {
		logger.WithError(err).Errorf("error sending message to server at %s: %v", e.url, err)
		if err := e.conn.Close(); err != nil {
			logger.WithError(err).Errorf("error closing connection to server at %s after write error: %v", e.url, err)
		}
		e.connected = false
	}
}

// read incoming messages from the submit server. This function should be called only by a single goroutine
func (e *serverEndpoint) readLoop(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		wsMsgType, payload, err := e.conn.ReadMessage()
		if err != nil {
			e.mutex.Lock()
			if e.connected {
				logger.WithError(err).Errorf("error reading websocket message from server at %s", e.url)
				if err := e.conn.Close(); err != nil {
					logger.WithError(err).Errorf("error closing connection to server at %s after read error: %v", e.url, err)
				}
				e.connected = false
			}
			e.mutex.Unlock()
			return
		}
		if wsMsgType != websocket.BinaryMessage {
			logger.Warnf("invalid message sent from server at %s. websocket message is not a binary message (%d)", e.url, websocket.BinaryMessage)
			continue
		}
		msg, err := submitws.FromBinary(payload)
		if err != nil {
			logger.WithError(err).Warnf("invalid message sent from server at %s. Error parsing websocket message: %v", e.url, err)
			continue
		}
		if e.handlers[msg.Type] == nil {
			logger.WithError(err).Warnf("invalid message sent form server at %s. No handler for message with type == %s", e.url, msg.Type)
			continue
		}
		wg.Add(1)
		go e.handlers[msg.Type](msg.Payload, wg)
	}
}

// returns a boolean indicating if the agent is connected to the submit server or not
func (e *serverEndpoint) isConnected() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.connected
}

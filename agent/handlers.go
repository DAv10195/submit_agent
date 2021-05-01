package agent

import (
	submitws "github.com/DAv10195/submit_commons/websocket"
	"time"
)

func (a *Agent) handleKeepaliveResponse(_ []byte) {
	newReadDeadLine := time.Now().Add(time.Minute)
	if err := a.endpoint.conn.SetReadDeadline(newReadDeadLine); err != nil {
		logger.WithError(err).Errorf("error advancing read deadline to %v on connection to %s", newReadDeadLine.Format(time.RFC3339), a.endpoint.url)
	}
	logger.Debugf("keepalive response received and read deadline advanced to %s on connection to %s", newReadDeadLine.Format(time.RFC3339), a.endpoint.url)
}

func (a *Agent) handlers() map[string]serverMessageHandler {
	handlers := make(map[string]serverMessageHandler)
	handlers[submitws.MessageTypeKeepaliveResponse] = a.handleKeepaliveResponse
	return handlers
}

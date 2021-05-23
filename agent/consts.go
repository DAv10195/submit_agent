package agent

import "time"

const (
	connTimeout				= 15 * time.Second

	connInterval			= 30 * time.Second

	keepaliveInterval		= 30 * time.Second

	agentIdFilePerms		= 0600

	googleDnsServer			= "8.8.8.8:80"

	udp						= "udp"

	authorization 			= "Authorization"

	encryptionKeyFileName	= "submit_agent.key"

	agentIdFileName			= "submit_agent.id"

	encryptedPrefix			= "encrypted:"

	flagServerPassword		= "submit-server-password"
	flagFileServerPassword	= "submit-file-server-password"

	queue					= "queue"
	maxMsgBatchFromQueue	= 10
	sendQueueInterval		= 3 * time.Second
)

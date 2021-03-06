package cmd

const (
	submit					= "submit"
	submitAgent 			= "submit_agent"
	start					= "start"

	defaultConfigFileName	= "submit_agent.yml"
	yaml					= "yaml"
	info					= "info"
	defMaxLogFileSize		= 10
	defMaxLogFileAge		= 3
	defMaxLogFileBackups	= 3
	deLogFileAndStdOut		= false
	defHost					= "localhost"
	defPort					= 8080
	defFsPort				= 8081
	defUser					= "admin"
	defPassword				= "admin"
	defMaxRunningTasks		= 10
	defMossParserHost		= "localhost"
	defMossParserPort		= 4567
	defMossPath				= "/usr/local/bin/moss"
	defSkipTlsVerify		= false
	defUseTls				= false

	flagConfigFile        	= "config-file"
	flagLogLevel          	= "log-level"
	flagLogFile           	= "log-file"
	flagLogFileAndStdout  	= "log-file-and-stdout"
	flagLogFileMaxSize    	= "log-file-max-size"
	flagLogFileMaxBackups 	= "log-file-max-backups"
	flagLogFileMaxAge     	= "log-file-max-age"

	flagCacheDir			= "cache-dir"

	flagServerHost			= "submit-server-host"
	flagServerPort			= "submit-server-port"
	flagServerUser			= "submit-server-user"
	flagServerPassword		= "submit-server-password"
	flagFileServerHost		= "submit-file-server-host"
	flagFileServerPort		= "submit-file-server-port"
	flagFileServerUser		= "submit-file-server-user"
	flagFileServerPassword	= "submit-file-server-password"

	flagMaxRunningTasks		= "max-running-tasks"

	flagMossParserHost		= "moss-parser-host"
	flagMossParserPort		= "moss-parser-port"
	flagMossPath			= "moss-path"

	flagTrustedCaFile		= "trusted-ca-file"
	flagSkipTlsVerify		= "skip-tls-verify"
	flagUseTls				= "use-tls"
)

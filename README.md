# Submit Agent

## An agent which subscribes to Submit Server Tasks Framework Messages and executes tasks sent from it, reporting back the results

### Usage:

```
submit_agent

Usage:
  submit_agent [command]

Available Commands:
  help        Help about any command
  start       start submit_agent

Flags:
  -h, --help   help for submit_agent

Use "submit_agent [command] --help" for more information about a command.
```

```
start submit_agent

Usage:
  submit_agent start [flags]

Flags:
      --cache-dir string                     path to the cache dir which will be used by the submit agent (default "/var/cache/submit-agent")
  -c, --config-file string                   path to submit agent config file
  -h, --help                                 help for start
      --log-file string                      log to file, specify the file location
      --log-file-and-stdout                  write logs to stdout if log-file is specified?
      --log-file-max-age int                 maximum age of the log file before it's rotated (default 3)
      --log-file-max-backups int             maximum number of log file rotations (default 3)
      --log-file-max-size int                maximum size of the log file before it's rotated (default 10)
      --log-level string                     logging level [panic, fatal, error, warn, info, debug] (default "info")
      --max-running-tasks int                max number of running tasks allowed to run in parallel (set to <= 0 for no limit) (default 10)
      --moss-parser-host string              moss parser host (default "localhost")
      --moss-parser-port int                 moss parser port (default 4567)
      --moss-path string                     moss path (default "/usr/local/bin/moss")
      --skip-tls-verify                      skip tls verification
      --submit-file-server-host string       submit file server hostname (or ip address) (default "localhost")
      --submit-file-server-password string   password to be used when authenticating against submit file server (default "admin")
      --submit-file-server-port int          submit file server port (default 8081)
      --submit-file-server-user string       user to be used when authenticating against submit file server (default "admin")
      --submit-server-host string            submit server hostname (or ip address) (default "localhost")
      --submit-server-password string        password to be used when authenticating against submit server (default "admin")
      --submit-server-port int               submit server port (default 8080)
      --submit-server-user string            user to be used when authenticating against submit server (default "admin")
      --trusted-ca-file string               trusted ca bundle path
      --use-tls                              use tls
```
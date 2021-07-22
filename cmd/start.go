package cmd

import (
	"context"
	"fmt"
	"github.com/DAv10195/submit_agent/agent"
	"github.com/DAv10195/submit_agent/path"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

func newStartCommand(ctx context.Context, args []string) *cobra.Command {
	var setupErr error
	var configFilePath string
	startCmd := &cobra.Command{
		Use: start,
		Short: fmt.Sprintf("%s %s", start, submitAgent),
		SilenceUsage: true,
		SilenceErrors: true,
		RunE: func (cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return err
			}
			if setupErr != nil {
				return setupErr
			}
			logLevel := viper.GetString(flagLogLevel)
			level, err := logrus.ParseLevel(logLevel)
			if err != nil {
				return err
			}
			logrus.SetLevel(level)
			logFile := viper.GetString(flagLogFile)
			if logFile != "" {
				lumberjackLogger := &lumberjack.Logger{
					Filename:   viper.GetString(flagLogFile),
					MaxSize:    viper.GetInt(flagLogFileMaxSize),
					MaxBackups: viper.GetInt(flagLogFileMaxBackups),
					MaxAge:     viper.GetInt(flagLogFileMaxAge),
					LocalTime:  true,
				}
				if viper.GetBool(flagLogFileAndStdout) {
					logrus.SetOutput(io.MultiWriter(os.Stdout, lumberjackLogger))
				} else {
					logrus.SetOutput(lumberjackLogger)
				}
			} else {
				logger.Debug("log file undefined")
			}
			cfg := &agent.Config{}
			cfg.SubmitServerHost = viper.GetString(flagServerHost)
			cfg.SubmitServerPort = viper.GetInt(flagServerPort)
			cfg.SubmitServerUser = viper.GetString(flagServerUser)
			cfg.SubmitServerPassword = viper.GetString(flagServerPassword)
			cfg.SubmitFsHost = viper.GetString(flagFileServerHost)
			cfg.SubmitFsPort = viper.GetInt(flagFileServerPort)
			cfg.SubmitFsUser = viper.GetString(flagFileServerUser)
			cfg.SubmitFsPassword = viper.GetString(flagFileServerPassword)
			cfg.CacheDir = viper.GetString(flagCacheDir)
			cfg.MaxRunningTasks = viper.GetInt(flagMaxRunningTasks)
			cfg.MossParserHost = viper.GetString(flagMossParserHost)
			cfg.MossParserPort = viper.GetInt(flagMossParserPort)
			cfg.MossPath = viper.GetString(flagMossPath)
			cfg.TrustedCaFilePath = viper.GetString(flagTrustedCaFile)
			cfg.SkipTlsVerify = viper.GetBool(flagSkipTlsVerify)
			cfg.UseTls = viper.GetBool(flagUseTls)
			if cfg.MaxRunningTasks <= 0 {
				logger.Warn("max running tasks is <= 0. No Limit will be imposed on the number of tasks executed in parallel")
			}
			cfg.ConfFile = configFilePath
			submitAgent, err := agent.NewAgent(cfg)
			if err != nil {
				return err
			}
			agentCtx, cancelAgentCtx := context.WithCancel(ctx)
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go submitAgent.Run(agentCtx, wg)
			logger.Info("agent is running")
			<- ctx.Done()
			logger.Info("stopping agent...")
			cancelAgentCtx()
			wg.Wait()
			logger.Info("agent stopped")
			return nil
		},
	}
	configFlagSet := pflag.NewFlagSet(submit, pflag.ContinueOnError)
	_ = configFlagSet.StringP(flagConfigFile, "c", "", "path to submit agent config file")
	configFlagSet.SetOutput(ioutil.Discard)
	_ = configFlagSet.Parse(args[1:])
	configFilePath, _ = configFlagSet.GetString(flagConfigFile)
	if configFilePath == "" {
		configFilePath = filepath.Join(path.GetDefaultConfigDirPath(), defaultConfigFileName)
	}
	viper.SetConfigType(yaml)
	viper.SetConfigFile(configFilePath)
	viper.SetDefault(flagLogFileAndStdout, deLogFileAndStdOut)
	viper.SetDefault(flagLogFileMaxSize, defMaxLogFileSize)
	viper.SetDefault(flagLogFileMaxAge, defMaxLogFileAge)
	viper.SetDefault(flagLogFileMaxBackups, defMaxLogFileBackups)
	viper.SetDefault(flagLogLevel, info)
	viper.SetDefault(flagServerHost, defHost)
	viper.SetDefault(flagServerPort, defPort)
	viper.SetDefault(flagServerUser, defUser)
	viper.SetDefault(flagServerPassword, defPassword)
	viper.SetDefault(flagFileServerHost, defHost)
	viper.SetDefault(flagFileServerPort, defFsPort)
	viper.SetDefault(flagFileServerUser, defUser)
	viper.SetDefault(flagFileServerPassword, defPassword)
	viper.SetDefault(flagCacheDir, path.GetDefaultCacheDirPath())
	viper.SetDefault(flagMaxRunningTasks, defMaxRunningTasks)
	viper.SetDefault(flagMossParserHost, defMossParserHost)
	viper.SetDefault(flagMossParserPort, defMossParserPort)
	viper.SetDefault(flagMossPath, defMossPath)
	viper.SetDefault(flagSkipTlsVerify, defSkipTlsVerify)
	viper.SetDefault(flagUseTls, defUseTls)
	startCmd.Flags().AddFlagSet(configFlagSet)
	startCmd.Flags().Int(flagLogFileMaxBackups, viper.GetInt(flagLogFileMaxBackups), "maximum number of log file rotations")
	startCmd.Flags().Int(flagLogFileMaxSize, viper.GetInt(flagLogFileMaxSize), "maximum size of the log file before it's rotated")
	startCmd.Flags().Int(flagLogFileMaxAge, viper.GetInt(flagLogFileMaxAge), "maximum age of the log file before it's rotated")
	startCmd.Flags().Bool(flagLogFileAndStdout, viper.GetBool(flagLogFileAndStdout), "write logs to stdout if log-file is specified?")
	startCmd.Flags().String(flagLogLevel, viper.GetString(flagLogLevel), "logging level [panic, fatal, error, warn, info, debug]")
	startCmd.Flags().String(flagLogFile, viper.GetString(flagLogFile), "log to file, specify the file location")
	startCmd.Flags().String(flagServerHost, viper.GetString(flagServerHost), "submit server hostname (or ip address)")
	startCmd.Flags().Int(flagServerPort, viper.GetInt(flagServerPort), "submit server port")
	startCmd.Flags().String(flagServerUser, viper.GetString(flagServerUser), "user to be used when authenticating against submit server")
	startCmd.Flags().String(flagServerPassword, viper.GetString(flagServerPassword), "password to be used when authenticating against submit server")
	startCmd.Flags().String(flagFileServerHost, viper.GetString(flagFileServerHost), "submit file server hostname (or ip address)")
	startCmd.Flags().Int(flagFileServerPort, viper.GetInt(flagFileServerPort), "submit file server port")
	startCmd.Flags().String(flagFileServerUser, viper.GetString(flagFileServerUser), "user to be used when authenticating against submit file server")
	startCmd.Flags().String(flagFileServerPassword, viper.GetString(flagFileServerPassword), "password to be used when authenticating against submit file server")
	startCmd.Flags().String(flagCacheDir, viper.GetString(flagCacheDir), "path to the cache dir which will be used by the submit agent")
	startCmd.Flags().Int(flagMaxRunningTasks, viper.GetInt(flagMaxRunningTasks), "max number of running tasks allowed to run in parallel (set to <= 0 for no limit)")
	startCmd.Flags().String(flagMossParserHost, viper.GetString(flagMossParserHost), "moss parser host")
	startCmd.Flags().Int(flagMossParserPort, viper.GetInt(flagMossParserPort), "moss parser port")
	startCmd.Flags().String(flagMossPath, viper.GetString(flagMossPath), "moss path")
	startCmd.Flags().Bool(flagSkipTlsVerify, viper.GetBool(flagSkipTlsVerify), "skip tls verification")
	startCmd.Flags().String(flagTrustedCaFile, viper.GetString(flagTrustedCaFile), "trusted ca bundle path")
	startCmd.Flags().Bool(flagUseTls, viper.GetBool(flagUseTls), "use tls")
	if err := viper.ReadInConfig(); err != nil && !os.IsNotExist(err) {
		setupErr = err
	}
	return startCmd
}

package main

import (
	"context"
	"github.com/DAv10195/submit_agent/cmd"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// set JSON logging format for all loggers of all packages
	logrus.SetFormatter(&logrus.JSONFormatter{})
	// get a temporary logger for the main function
	logger := logrus.WithFields(logrus.Fields{"component":"main"})
	// create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// register to os signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	// listen to signals
	go func() {
		defer cancel()
		logger.Infof("signal received: %v", <-signalChan)
	}()
	// create the root cmd and execute the given command
	rootCmd := cmd.NewRootCmd(ctx, os.Args)
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Fatal("error running submit agent")
	}
}

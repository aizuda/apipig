package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	remoteAgent "apipig/app/apps/remote-agent/agent"
)

func main() {
	configPath := flag.String("c", "remote-agent.yaml", "remote agent configuration file")
	flag.Parse()
	if err := remoteAgent.EnsureNonRoot(); err != nil {
		log.Fatal(err)
	}
	config, err := remoteAgent.LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	logger, closeLog, err := newLogger(config.LogFile)
	if err != nil {
		log.Fatal(err)
	}
	defer closeLog()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	runner := remoteAgent.NewRunner(config, logger)
	if err := runner.Run(ctx); err != nil && err != context.Canceled {
		logger.Fatal(err)
	}
}

func newLogger(path string) (*log.Logger, func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, func() {}, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, func() {}, err
	}
	logger := log.New(io.MultiWriter(os.Stdout, file), "remote-agent ", log.Ldate|log.Ltime|log.LUTC)
	return logger, func() {
		if closeErr := file.Close(); closeErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, closeErr)
		}
	}, nil
}

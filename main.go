package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/adamdecaf/deadcheck/internal/api"
	"github.com/adamdecaf/deadcheck/internal/check"
	"github.com/adamdecaf/deadcheck/internal/config"
	"github.com/moov-io/base/log"
)

var (
	flagConfig   = flag.String("config", "", "Filepath to configuration file")
	flagHttpAddr = flag.String("http.addr", ":8080", "HTTP listen address")
	flagVersion  = flag.Bool("version", false, "Print the version of deadcheck")
)

func main() {
	flag.Parse()

	if *flagVersion {
		fmt.Printf("deadcheck %s", Version) //nolint:forbidigo
		return
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	logger := log.NewDefaultLogger().With(log.Fields{
		"app":     log.String("deadcheck"),
		"version": log.String(Version),
	})

	conf, err := config.Load(*flagConfig)
	if err != nil {
		logger.Error().LogErrorf("reading %s failed: %v", *flagConfig, err)
		os.Exit(1)
	}

	instances, err := check.Setup(ctx, logger, conf)
	if err != nil {
		logger.Error().LogErrorf("setting up checks failed: %w", err)
		os.Exit(1)
	}

	server, err := api.Server(logger, *flagHttpAddr, instances)
	if err != nil {
		logger.Error().LogErrorf("running HTTP server failed: %w", err)
		os.Exit(1)
	}
	defer func() {
		if server != nil {
			server.Shutdown(ctx)
		}
	}()

	// Listen for shutdown
	errs := make(chan error)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("signal: %v", <-c)
	}()

	err = <-errs
	if err != nil {
		logger.Warn().Logf("shutting down: %v", err)
	}
}

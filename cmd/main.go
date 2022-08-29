package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/server"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{Level: hclog.Debug, Color: hclog.AutoColor})
	s, err := server.New(logger)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	go func() {
		err := s.Start()
		if err != nil {
			logger.Error("Start server error", "error", err)
			os.Exit(1)
		}
	}()

	// trap sigterm or interupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Block until a signal is received.
	sig := <-c
	logger.Info("Graceful shutdown", "signal:", sig)

	err = s.Shutdown()
	if err != nil {
		logger.Error("Unable to shutdown server", "error", err)
	}
}

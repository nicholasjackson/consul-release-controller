package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/env"
)

var enableKubernetes = env.Bool("ENABLE_KUBERNETES", false, false, "Should Kubernetes integration be enabled")
var enableNomad = env.Bool("ENABLE_NOMAD", false, true, "Should Nomad integration be enabled")
var enableHTTP = env.Bool("ENABLE_HTTP", false, false, "Should the server listen on port 8080 plain HTTP")

func main() {
	env.Parse()

	logger := hclog.New(&hclog.LoggerOptions{Level: hclog.Debug, Color: hclog.AutoColor})
	s, err := New(logger, *enableKubernetes, *enableNomad, *enableHTTP)
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

	//logger.Info("Restarting server")
	//go func() {
	//	err = s.Start()
	//	if err != nil {
	//		logger.Error("Unable to restart server", "error", err)
	//		os.Exit(1)
	//	}
	//}()

	//sig = <-c
	//logger.Info("Graceful shutdown", "signal:", sig)

	//err = s.Shutdown()
	//if err != nil {
	//	logger.Error("Unable to shutdown server", "error", err)
	//}
}

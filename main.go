package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nicholasjackson/consul-canary-controller/controller"
)

func main() {
	s := controller.New()

	go func() {
		err := s.Start()
		if err != nil {
			os.Exit(1)
		}
	}()

	// trap sigterm or interupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Block until a signal is received.
	sig := <-c
	fmt.Println("Graceful shutdown, got signal:", sig)

	s.Shutdown()
}

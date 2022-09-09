package api

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/api/handlers"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
)

// Server is the main HTTP API server
type Server struct {
	httpsServer   *http.Server
	httpServer    *http.Server
	tlsConfig     *tls.Config
	logger        hclog.Logger
	router        chi.Router
	config        *ServerConfig
	httpsListener net.Listener
	httpListener  net.Listener
	doneChan      chan struct{}
}

// ServerConfig defines the configuration for the APIServer
type ServerConfig struct {
	TLSBindAddress  string
	TLSBindPort     int
	HTTPBindAddress string
	HTTPBindPort    int
	TLSCertLocation string
	TLSKeyLocation  string
}

// New creates a new APIServer
func New(config *ServerConfig, p interfaces.Provider, l hclog.Logger) (*Server, error) {
	apiServer := &Server{logger: l, config: config, doneChan: make(chan struct{})}

	healthHandler := handlers.NewHealthHandlers(l.Named("health-handlers"))
	apiHandler := handlers.NewReleaseHandler(p)

	httplogger := httplog.NewLogger("http-server")
	httplogger = httplogger.Output(hclog.NewNullLogger().StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Trace}))

	rtr := chi.NewRouter()
	rtr.Use(httplog.RequestLogger(httplogger))

	// add health and ready endpoints
	rtr.Get("/v1/health", healthHandler.Health)
	rtr.Get("/v1/ready", healthHandler.Ready)

	// configure the main API
	rtr.Post("/v1/releases", apiHandler.Post)
	rtr.Get("/v1/releases", apiHandler.GetAll)
	rtr.Get("/v1/releases/{name}", apiHandler.GetSingle)
	rtr.Delete("/v1/releases/{name}", apiHandler.Delete)

	apiServer.router = rtr

	certificate, err := tls.LoadX509KeyPair(config.TLSCertLocation, config.TLSKeyLocation)
	if err != nil {
		return nil, fmt.Errorf("unable to load TLS certificates:	%s", err)
	}

	apiServer.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{certificate},
		Rand:         rand.Reader,
	}

	return apiServer, nil
}

// Start the server and block until complete
func (a *Server) Start() error {
	errChan := make(chan error)

	if a.config.TLSBindAddress != "" {
		// Create TLS listener.
		a.logger.Info("HTTPS Listening on ", "address", a.config.TLSBindAddress, "port", a.config.TLSBindPort)
		l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", a.config.TLSBindAddress, a.config.TLSBindPort))
		if err != nil {
			return fmt.Errorf("unable to create TCP listener: %s", err)
		}

		a.httpsListener = l

		a.httpsServer = &http.Server{
			Handler:      a.router,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		// start the TLS endpoint
		go func() {
			err := a.httpsServer.Serve(tls.NewListener(l, a.tlsConfig))
			if err != nil && err != http.ErrServerClosed {
				errChan <- fmt.Errorf("unable to start TLS server: %s", err)
			}
		}()
	}

	// if we are listening on plain HTTP start the server
	if a.config.HTTPBindAddress != "" {
		l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", a.config.HTTPBindAddress, a.config.HTTPBindPort))
		if err != nil {
			errChan <- fmt.Errorf("unable to create TCP listener: %s", err)
		}

		a.httpListener = l

		a.logger.Info("HTTP Listening on ", "address", a.config.HTTPBindAddress, "port", a.config.HTTPBindPort)
		a.httpServer = &http.Server{
			Addr:         fmt.Sprintf("%s:%d", a.config.HTTPBindAddress, a.config.HTTPBindPort),
			Handler:      a.router,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		go func() {
			err := a.httpServer.Serve(l)
			if err != nil && err != http.ErrServerClosed {
				errChan <- fmt.Errorf("unable to start HTTP server: %s", err)
			}
		}()
	}

	select {
	case err := <-errChan:
		return err
	case <-a.doneChan:
		return nil
	}
}

// Shutdown the server and close all connections
func (a *Server) Shutdown() error {
	var httpsFile *os.File
	var httpFile *os.File
	var err error

	if a.httpsListener != nil {
		httpsFile, err = a.httpsListener.(*net.TCPListener).File()
		if err != nil {
			return fmt.Errorf("unable to get file for HTTPS listener: %s", err)
		}
	}

	if a.httpListener != nil {
		httpsFile, err = a.httpListener.(*net.TCPListener).File()
		if err != nil {
			return fmt.Errorf("unable to get file for HTTPS listener: %s", err)
		}
	}

	if a.httpsServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err = a.httpsServer.Shutdown(ctx)

		if err != nil {
			return fmt.Errorf("unable to shutdown HTTPS server: %s", err)
		}
	}

	if a.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := a.httpServer.Shutdown(ctx)

		if err != nil {
			return fmt.Errorf("unable to shutdown HTTP server: %s", err)
		}
	}

	// close the listener for the HTTPS server
	if httpsFile != nil {
		err = httpsFile.Close()
		if err != nil {
			return fmt.Errorf("unable to shutdown HTTPS listener: %s", err)
		}
	}

	// close the listener for the HTTP server
	if httpFile != nil {
		err = httpFile.Close()
		if err != nil {
			return fmt.Errorf("unable to shutdown HTTP listener: %s", err)
		}
	}

	return nil
}

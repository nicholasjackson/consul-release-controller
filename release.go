package main

import (
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
	"github.com/nicholasjackson/consul-release-controller/config"
	"github.com/nicholasjackson/consul-release-controller/handlers/api"
	kubernetes "github.com/nicholasjackson/consul-release-controller/kubernetes/controller"
	nomad "github.com/nicholasjackson/consul-release-controller/nomad/controller"
	"github.com/nicholasjackson/consul-release-controller/plugins"
	"github.com/nicholasjackson/consul-release-controller/plugins/consul"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/plugins/prometheus"
	"golang.org/x/net/context"
)

type Release struct {
	log                  hclog.Logger
	server               *http.Server
	httpServer           *http.Server
	listener             net.Listener
	metrics              *prometheus.Metrics
	kubernetesController *kubernetes.Kubernetes
	nomadController      *nomad.Nomad
	enableKubernetes     bool
	enableNomad          bool
	tlsBindAddress       string
	tlsBindPort          int
	httpBindAddress      string
	httpBindPort         int
	shutdown             chan struct{}
}

func New(log hclog.Logger) (*Release, error) {
	metrics, err := prometheus.NewMetrics(config.MetricsBindAddress(), config.MetricsPort(), "/metrics")
	if err != nil {
		log.Error("failed to create metrics", "error", err)
		return nil, err
	}

	return &Release{
		log:              log,
		metrics:          metrics,
		enableKubernetes: config.EnableKubernetes(),
		enableNomad:      config.EnableNomad(),
		tlsBindAddress:   config.TLSAPIBindAddress(),
		tlsBindPort:      config.TLSAPIPort(),
		httpBindAddress:  config.HTTPAPIBindAddress(),
		httpBindPort:     config.HTTPAPIPort(),
		shutdown:         make(chan struct{}),
	}, nil
}

// Start the server and block until exit
func (r *Release) Start() error {
	// start the prometheus metrics server
	r.metrics.StartServer()
	r.metrics.ServiceStarting()

	//store := memory.NewStore()
	store, err := consul.NewStorage(r.log.Named("releaser-plugin-consul"))
	if err != nil {
		r.log.Error("failed to create storage", "error", err)
		return err
	}

	provider := plugins.GetProvider(r.log, r.metrics, store)

	// reload any releases that are currently in process, the controller may have crashed part way
	// through an operation.
	rehydrateReleases(provider, r.log)

	if r.enableKubernetes {
		// create the kubernetes controller
		kc := kubernetes.New(provider, config.TLSCertificate(), config.TLSKey(), config.KubernetesControllerPort())
		r.kubernetesController = kc
		go kc.Start()
	}

	if r.enableNomad {
		// create the kubernetes controller
		nc, err := nomad.New(provider)
		if err != nil {
			return fmt.Errorf("Unable to create Nomad controller: %s", err)
		}

		r.nomadController = nc
		go nc.Start()
	}

	healthHandler := api.NewHealthHandlers(r.log.Named("health-handlers"))
	apiHandler := api.NewReleaseHandler(provider)

	r.log.Info("Starting controller")
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

	certificate, err := tls.LoadX509KeyPair(config.TLSCertificate(), config.TLSKey())
	if err != nil {
		r.log.Error("Error loading certificates", "error", err)
		return fmt.Errorf("unable to load TLS certificates:	%s", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		Rand:         rand.Reader,
	}

	// Create TLS listener.
	r.log.Info("Listening on ", "address", r.tlsBindAddress, "port", r.tlsBindPort)
	l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", r.tlsBindAddress, r.tlsBindPort))
	if err != nil {
		r.log.Error("Error creating TCP listener", "error", err)
		return fmt.Errorf("unable to create TCP listener: %s", err)
	}

	r.listener = l

	r.server = &http.Server{
		Handler:      rtr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		err := r.server.Serve(tls.NewListener(l, config))
		if err != nil && err != http.ErrServerClosed {
			r.log.Error("Unable to start TLS server", "error", err)
		}
	}()

	if r.httpBindAddress != "" {
		l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", r.httpBindAddress, r.httpBindPort))
		if err != nil {
			r.log.Error("Error creating TCP listener", "error", err)
			return fmt.Errorf("unable to create TCP listener: %s", err)
		}

		r.log.Info("Listening on ", "address", r.httpBindAddress, "port", r.httpBindPort)
		r.httpServer = &http.Server{
			Addr:         fmt.Sprintf("%s:%d", r.httpBindAddress, r.httpBindPort),
			Handler:      rtr,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		go func() {
			err := r.httpServer.Serve(l)
			if err != nil && err != http.ErrServerClosed {
				r.log.Error("Unable to start HTTP server", "error", err)
			}
		}()
	}

	<-r.shutdown

	return nil
}

// Shutdown the server gracefully
func (r *Release) Shutdown() error {
	r.log.Info("Shutting down server gracefully")

	// to reuse the listener the listeners file must be closed
	var lnFile *os.File
	var err error

	if r.listener != nil {
		lnFile, err = r.listener.(*net.TCPListener).File()
		if err != nil {
			r.log.Error("Unable to get file for listener", "error", err)
			return err
		}
	}

	if r.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err = r.server.Shutdown(ctx)

		if err != nil {
			r.log.Error("Unable to shutdown server", "error", err)
			return err
		}
	}

	if r.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err = r.httpServer.Shutdown(ctx)

		if err != nil {
			r.log.Error("Unable to shutdown server", "error", err)
			return err
		}
	}

	// close the listener for the server
	if lnFile != nil {
		r.log.Info("Shutting down listener")
		err = lnFile.Close()
		if err != nil {
			r.log.Error("Unable to shutdown listener", "error", err)
			return err
		}
	}

	if r.metrics != nil {
		r.log.Info("Shutting down metrics")
		err = r.metrics.StopServer()
		if err != nil {
			r.log.Error("Unable to shutdown metrics", "error", err)
			return err
		}
	}

	if r.kubernetesController != nil {
		r.log.Info("Shutting down Kubernetes controller")
		r.kubernetesController.Stop()
	}

	if r.nomadController != nil {
		r.log.Info("Shutting down Nomad controller")
		r.nomadController.Stop()
	}

	r.shutdown <- struct{}{}

	return nil
}

func rehydrateReleases(p interfaces.Provider, logger hclog.Logger) {
	s := p.GetDataStore()

	rels, err := s.ListReleases(&interfaces.ListOptions{})
	if err != nil {
		logger.Error("Unable to list releases", "error", err)
		return
	}

	for _, r := range rels {
		logger.Info("Rehydrating release", "name", r.Name, "state", r.CurrentState())
		sm, err := p.GetStateMachine(r)
		if err != nil {
			logger.Error("Unable to list releases", "error", err)
		}

		go sm.Resume()
	}

	//panic("exit")
}

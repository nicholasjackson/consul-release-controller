package controller

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
	"github.com/nicholasjackson/consul-release-controller/plugins"
	"github.com/nicholasjackson/consul-release-controller/plugins/consul"
	"github.com/nicholasjackson/consul-release-controller/plugins/prometheus"
	"golang.org/x/net/context"
)

type Release struct {
	log                  hclog.Logger
	server               *http.Server
	listener             net.Listener
	metrics              *prometheus.Metrics
	kubernetesController *kubernetes.Kubernetes
}

func New(log hclog.Logger) (*Release, error) {
	metrics, err := prometheus.NewMetrics(config.MetricsBindAddress(), config.MetricsPort(), "/metrics")
	if err != nil {
		log.Error("failed to create metrics", "error", err)
		return nil, err
	}

	return &Release{log: log, metrics: metrics}, nil
}

// Start the server and block until exit
func (r *Release) Start() error {
	// start the prometheus metrics server
	r.metrics.StartServer()
	r.metrics.ServiceStarting()

	//store := memory.NewStore()
	store, _ := consul.NewStorage(r.log.Named("releaser-plugin-consul"))
	provider := plugins.GetProvider(r.log, r.metrics, store)

	// create the kubernetes controller
	kc := kubernetes.New(provider, config.TLSCertificate(), config.TLSKey(), config.KubernetesControllerPort())
	r.kubernetesController = kc
	go kc.Start()

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
	l, err := net.Listen("tcp", ":9443")
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

	err = r.server.Serve(tls.NewListener(l, config))
	if err != http.ErrServerClosed {
		return fmt.Errorf("unable to start server: %s", err)
	}

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
		r.log.Info("Shutting down kubernetes controller")
		r.kubernetesController.Stop()
	}

	return nil
}

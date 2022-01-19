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
	"github.com/nicholasjackson/consul-release-controller/handlers/api"
	kubernetes "github.com/nicholasjackson/consul-release-controller/kubernetes/controller"
	promMetrics "github.com/nicholasjackson/consul-release-controller/metrics"
	"github.com/nicholasjackson/consul-release-controller/plugins"
	"github.com/nicholasjackson/consul-release-controller/state"
	"golang.org/x/net/context"
)

type Release struct {
	log                  hclog.Logger
	server               *http.Server
	listener             net.Listener
	metrics              *promMetrics.Sink
	kubernetesController *kubernetes.Kubernetes
}

func New(log hclog.Logger) (*Release, error) {
	metrics, err := promMetrics.New("0.0.0.0", 9102, "/metrics")
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

	store := state.NewInmemStore()

	k8sHandler, _ := kubernetes.NewK8sWebhook(r.log.Named("kubernetes-webhook"), r.metrics, store, plugins.GetProvider(r.log))
	healthHandler := api.NewHealthHandlers(r.log.Named("health-handlers"))
	apiHandler := api.NewReleaseHandler(r.log.Named("restful-api"), r.metrics, store, plugins.GetProvider(r.log))

	r.log.Info("Starting controller")
	httplogger := httplog.NewLogger("http-server")
	httplogger = httplogger.Output(r.log.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Trace}))

	rtr := chi.NewRouter()
	rtr.Use(httplog.RequestLogger(httplogger))

	// add health and ready endpoints
	rtr.Get("/v1/health", healthHandler.Health)
	rtr.Get("/v1/ready", healthHandler.Ready)

	// configure kubernetes webhooks
	rtr.Post("/k8s/mutating", k8sHandler.Mutating())

	// configure the main API
	rtr.Post("/v1/releases", apiHandler.Post)
	rtr.Get("/v1/releases", apiHandler.Get)
	rtr.Delete("/v1/releases/{name}", apiHandler.Delete)

	certificate, err := tls.LoadX509KeyPair(os.Getenv("TLS_CERT"), os.Getenv("TLS_KEY"))
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
	lnFile, err := r.listener.(*net.TCPListener).File()
	if err != nil {
		r.log.Error("Unable to get file for listener", "error", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = r.server.Shutdown(ctx)

	if err != nil {
		r.log.Error("Unable to shutdown server", "error", err)
		return err
	}

	// close the listener for the server
	r.log.Info("Shutting down listener")
	err = lnFile.Close()
	if err != nil {
		r.log.Error("Unable to shutdown listener", "error", err)
		return err
	}

	r.log.Info("Shutting down metrics")
	err = r.metrics.StopServer()
	if err != nil {
		r.log.Error("Unable to shutdown metrics", "error", err)
		return err
	}

	return nil
}

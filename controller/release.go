package controller

import (
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/handlers/api"
	"github.com/nicholasjackson/consul-canary-controller/kubernetes"
	promMetrics "github.com/nicholasjackson/consul-canary-controller/metrics"
	"github.com/nicholasjackson/consul-canary-controller/plugins"
	"github.com/nicholasjackson/consul-canary-controller/state"
	"golang.org/x/net/context"
)

type Release struct {
	log    hclog.Logger
	server *http.Server
}

func New() *Release {
	log := hclog.New(&hclog.LoggerOptions{Color: hclog.AutoColor, Level: hclog.Debug})
	return &Release{log: log}
}

// Start the server and block until exit
func (r *Release) Start() error {
	metrics, err := promMetrics.New("0.0.0.0", 9102, "/metrics")
	if err != nil {
		r.log.Error("failed to create metrics", "error", err)
		return err
	}

	// start the prometheus metrics server
	metrics.StartServer()
	metrics.ServiceStarting()

	store := state.NewInmemStore()

	k8sHandler, _ := kubernetes.NewK8sWebhook(r.log.Named("kubernetes-webhook"), metrics, store, plugins.GetProvider())
	healthHandler := api.NewHealthHandlers(r.log.Named("health-handlers"))
	apiHandler := api.NewReleaseHandler(r.log.Named("restful-api"), metrics, store, plugins.GetProvider())

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

	r.server = &http.Server{
		Addr:    ":9443",
		Handler: rtr,
	}

	err = r.server.ListenAndServeTLS(os.Getenv("TLS_CERT"), os.Getenv("TLS_KEY"))
	if err != nil && err != http.ErrServerClosed {
		r.log.Error("error starting server", "error", err)
		return err
	}

	return nil
}

// Shutdown the server gracefully
func (r *Release) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return r.server.Shutdown(ctx)
}

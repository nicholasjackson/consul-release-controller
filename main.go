package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/handlers/api"
	promMetrics "github.com/nicholasjackson/consul-canary-controller/metrics"
	"github.com/nicholasjackson/consul-canary-controller/plugins"
	"github.com/nicholasjackson/consul-canary-controller/state"
)

func main() {
	log := hclog.Default()
	log.SetLevel(hclog.Debug)

	metrics, err := promMetrics.New("0.0.0.0", 9102, "/metrics")
	if err != nil {
		log.Error("failed to create metrics", "error", err)
		os.Exit(1)
		return
	}

	// start the prometheus metrics server
	metrics.StartServer()

	metrics.ServiceStarting()

	//k8sHandler, _ := kubernetes.NewK8sWebhook(log, metrics, state.NewInmemStore(), plugins.GetProvider())
	healthHandler := api.NewHealthHandlers(log)
	apiHandler := api.NewReleaseHandler(log, metrics, state.NewInmemStore(), plugins.GetProvider())

	log.Info("Starting controller")
	httplogger := httplog.NewLogger("consul-canary")
	httplogger = httplogger.Output(log.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug}))

	r := chi.NewRouter()
	r.Use(httplog.RequestLogger(httplogger))

	// add health and ready endpoints
	r.Get("/v1/health", healthHandler.Health)
	r.Get("/v1/ready", healthHandler.Ready)

	// configure kubernetes webhooks
	r.Post("/k8s/mutating", func(rw http.ResponseWriter, r *http.Request) {
		d, _ := ioutil.ReadAll(r.Body)
		fmt.Println(string(d))
	})

	// configure the main API
	r.Post("/v1/deployments", apiHandler.Post)
	r.Get("/v1/deployments", apiHandler.Get)
	r.Delete("/v1/deployments/{name}", apiHandler.Delete)

	err = http.ListenAndServeTLS(":9443", "/tmp/k8s-webhook-server/serving-certs/tls.crt", "/tmp/k8s-webhook-server/serving-certs/tls.key", r)
	if err != http.ErrServerClosed {
		log.Error("error starting server", "error", err)
	}
}

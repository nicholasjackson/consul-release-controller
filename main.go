package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/handlers"
)

func main() {
	log := hclog.Default()
	log.SetLevel(hclog.Debug)

	k8sHandler, _ := handlers.NewK8sWebhook(log)

	log.Info("Starting controller")
	httplogger := httplog.NewLogger("consul-canary")
	httplogger = httplogger.Output(log.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug}))

	r := chi.NewRouter()
	r.Use(httplog.RequestLogger(httplogger))
	r.Post("/k8s/mutating", k8sHandler.Mutating())

	http.ListenAndServeTLS(":9443", "/tmp/k8s-webhook-server/serving-certs/tls.crt", "/tmp/k8s-webhook-server/serving-certs/tls.key", r)
}

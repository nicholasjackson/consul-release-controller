package handlers

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/go-hclog"
)

type HealthHandlers struct {
	log hclog.Logger
}

func NewHealthHandlers(l hclog.Logger) *HealthHandlers {
	return &HealthHandlers{l}
}

func (h *HealthHandlers) Health(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(rw, "OK")
}

func (h *HealthHandlers) Ready(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(rw, "OK")
}

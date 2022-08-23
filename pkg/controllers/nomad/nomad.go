package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/clients"
	controller "github.com/nicholasjackson/consul-release-controller/pkg/controllers"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
)

// Nomad defines a release controller for the Nomad scheduler
type Nomad struct {
	log         hclog.Logger
	nomadClient clients.Nomad
	ctx         context.Context
	cancel      context.CancelFunc
	admission   controller.Admission
}

// New returns a new Nomad release controller
func New(p interfaces.Provider) (*Nomad, error) {
	l := p.GetLogger().ResetNamed("nomad-admission")
	a := controller.NewAdmission(p, l)

	nc, err := clients.NewNomad(2*time.Second, 600*time.Second, p.GetLogger().ResetNamed("nomad-client"))
	if err != nil {
		return nil, err
	}

	return &Nomad{log: l, nomadClient: nc, admission: a}, nil
}

// Start the Nomad controller
func (n *Nomad) Start() error {
	n.log.Info("Starting controller, listening for deployment events")

	n.ctx, n.cancel = context.WithCancel(context.Background())
	// start monitoring the events feed
	events, err := n.nomadClient.GetEvents(n.ctx)
	if err != nil {
		n.log.Error("Error getting events from Nomad", "error", err)
		return fmt.Errorf("unable to register for Nomad events: %s", err)
	}

	for evts := range events {
		for _, evt := range evts.Events {
			n.log.Debug("Received new job event", "event", evt.Type, "topic", evt.Topic)

			// we are only interested in JobRegistered events
			switch evt.Type {
			case "JobRegistered":
				j, _ := evt.Job()
				if *j.Status == "pending" {
					n.log.Info("Handle Job registration", "name", *j.Name, "namespace", *j.Namespace)

					_, err := n.admission.Check(context.Background(), *j.Name, *j.Namespace, j.Meta, fmt.Sprintf("%d", *j.Version), interfaces.RuntimePlatformNomad)
					if err != nil {
						n.log.Error("Admission failed", "name", *j.Name, "namespace", *j.Namespace, "error", err)
						continue
					}

					n.log.Info("Admission succeeded", "name", *j.Name, "namespace", *j.Namespace, "error", err)
				}
			}
		}
	}

	n.log.Debug("Exit event loop")
	return nil
}

// Stop the Nomad controller
func (n *Nomad) Stop() {
	n.cancel()
}

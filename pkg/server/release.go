package server

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/api"
	"github.com/nicholasjackson/consul-release-controller/pkg/config"
	kubernetes "github.com/nicholasjackson/consul-release-controller/pkg/controllers/kubernetes"
	nomad "github.com/nicholasjackson/consul-release-controller/pkg/controllers/nomad"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/consul"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/prometheus"
)

type Release struct {
	log                  hclog.Logger
	metrics              *prometheus.Metrics
	kubernetesController *kubernetes.Kubernetes
	nomadController      *nomad.Nomad
	apiServer            *api.Server
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

	// increment the start metrics counter
	r.metrics.ServiceStarting()

	//store := memory.NewStore()
	store, err := consul.NewStorage(r.log.Named("releaser-plugin-consul"))
	if err != nil {
		r.log.Error("failed to create storage", "error", err)
		return err
	}

	provider := plugins.GetProvider(r.log, r.metrics, store)

	apiError := make(chan error)
	kubernetesError := make(chan error)
	nomadError := make(chan error)

	// reload any releases that are currently in process, the controller may have crashed part way
	// through an operation.
	err = rehydrateReleases(provider, r.log)
	if err != nil {
		return fmt.Errorf("Unable to rehydrate releases: %s", err)
	}

	if r.enableKubernetes {
		r.log.Info("Starting Kubernetes Controller")

		// create the kubernetes controller
		kc := kubernetes.New(provider, config.TLSCertificate(), config.TLSKey(), config.KubernetesControllerPort())
		r.kubernetesController = kc
		go func() {
			err := kc.Start()
			if err != nil {
				kubernetesError <- err
			}
		}()
	}

	if r.enableNomad {
		r.log.Info("Starting Nomad Controller")

		// create the kubernetes controller
		nc, err := nomad.New(provider)
		if err != nil {
			return fmt.Errorf("Unable to create Nomad controller: %s", err)
		}

		r.nomadController = nc
		go func() {
			err := nc.Start()
			if err != nil {
				kubernetesError <- err
			}
		}()
	}

	// create the API server
	c := &api.ServerConfig{
		TLSBindAddress:  config.TLSAPIBindAddress(),
		TLSBindPort:     config.TLSAPIPort(),
		HTTPBindAddress: config.HTTPAPIBindAddress(),
		HTTPBindPort:    config.HTTPAPIPort(),
		TLSCertLocation: config.TLSCertificate(),
		TLSKeyLocation:  config.TLSKey(),
	}

	as, err := api.New(c, provider, r.log.Named("api-server"))
	if err != nil {
		return fmt.Errorf("unable to create API server: %s", err)
	}

	r.log.Info("Starting API Server")
	r.apiServer = as

	go func() {
		err := r.apiServer.Start()
		if err != nil {
			apiError <- err
		}
	}()

	select {
	case <-r.shutdown:
		r.log.Debug("Shutdown received, start loop exiting")
	case err := <-kubernetesError:
		r.log.Error("Kubernetes error message received, start loop exiting", "error", err)
		return err
	case err := <-nomadError:
		r.log.Error("Nomad error message received, start loop exiting", "error", err)
		return err
	case err := <-apiError:
		r.log.Error("API error message received, start loop exiting", "error", err)
		return err
	}

	return nil
}

// Shutdown the server gracefully
func (r *Release) Shutdown() error {
	r.log.Info("Shutting down server gracefully")

	if r.apiServer != nil {
		r.log.Info("Shutting down API server")
		err := r.apiServer.Shutdown()
		if err != nil {
			r.log.Error("Unable to shutdown API server", "error", err)
			return fmt.Errorf("unable to shutdown API server: %s", err)
		}
	}

	if r.metrics != nil {
		r.log.Info("Shutting down metrics")
		err := r.metrics.StopServer()
		if err != nil {
			r.log.Error("Unable to shutdown metrics", "error", err)
			return fmt.Errorf("unable to shutdown metrics server: %s", err)
		}

		r.log.Debug("Metrics server stopped")
	}

	if r.kubernetesController != nil {
		r.log.Info("Shutting down Kubernetes controller")
		r.kubernetesController.Stop()
		r.log.Debug("Kubernetes controller stopped")
	}

	if r.nomadController != nil {
		r.log.Info("Shutting down Nomad controller")
		r.nomadController.Stop()
		r.log.Debug("Nomad controller stopped")
	}

	r.log.Debug("Shutdown complete")
	r.shutdown <- struct{}{}

	return nil
}

func rehydrateReleases(p interfaces.Provider, logger hclog.Logger) error {
	s := p.GetDataStore()

	rels, err := s.ListReleases(&interfaces.ListOptions{})
	if err != nil {
		return fmt.Errorf("unable to list releases: %s", err)
	}

	for _, r := range rels {
		logger.Info("Rehydrating release", "name", r.Name, "state", r.CurrentState())
		sm, err := p.GetStateMachine(r)
		if err != nil {
			return fmt.Errorf("unable to get statemachine for release: %s, %s", r.Name, err)
		}

		go sm.Resume()
	}

	return nil

	//panic("exit")
}

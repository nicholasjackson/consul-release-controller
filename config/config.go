package config

import (
	"os"
	"strconv"
)

// TLSCertificate returns the path to the TLS certificate used to secure the API and webhook transport
func TLSCertificate() string {
	return os.Getenv("TLS_CERT")
}

// TLSKey returns the path to the TLS key used to secure the API and webhook transport
func TLSKey() string {
	return os.Getenv("TLS_KEY")
}

// KubeConfig returns the path to the Kubernetes config that can be used to contact the Kubernetes
// API sever
func KubeConfig() string {
	return os.Getenv("KUBECONFIG")
}

// ConsulServiceUpstreams returns the URI of the Envoy proxy that is serving Service Mesh
// endpoints for services under test
func ConsulServiceUpstreams() string {
	return os.Getenv("UPSTREAMS")
}

func APIBindAddress() string {
	if a := os.Getenv("API_BIND_ADDRESS"); a != "" {
		return a
	}

	return "0.0.0.0"
}

func APIPort() int {
	if a := os.Getenv("API_PORT"); a != "" {
		p, err := strconv.Atoi(a)
		if err != nil {
			return p
		}
	}

	return 9443
}

func MetricsBindAddress() string {
	if a := os.Getenv("METRICS_BIND_ADDRESS"); a != "" {
		return a
	}

	return "0.0.0.0"
}

func MetricsPort() int {
	if a := os.Getenv("METRICS_PORT"); a != "" {
		p, err := strconv.Atoi(a)
		if err != nil {
			return p
		}
	}

	return 9102
}

func KubernetesControllerBindAddress() string {
	if a := os.Getenv("K8S_CONTROLLER_ADDRESS"); a != "" {
		return a
	}

	return "0.0.0.0"
}

func KubernetesControllerPort() int {
	if a := os.Getenv("K8S_CONTROLLER_PORT"); a != "" {
		p, err := strconv.Atoi(a)
		if err != nil {
			return p
		}
	}

	return 19443
}

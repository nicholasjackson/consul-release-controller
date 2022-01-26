# Mandatory varirables
variable "consul_k8s_cluster" {
  default = "dc1"
}

variable "consul_k8s_network" {
  default = "dc1"
}

variable "consul_monitoring_enabled" {
  description = "Should the monitoring stack, Prometheus, Grafana, Loki be installed"
  default     = true
}

variable "consul_ingress_gateway_enabled" {
  description = "Should the Ingress gateways be enabled"
  default     = true
}

variable "consul_acls_enabled" {
  description = "Enable ACLs for securing the Consul server"
  default     = true
}

variable "consul_image" {
  default = "hashicorp/consul:1.9.13"
}

variable "consul_envoy_image" {
  default     = "envoyproxy/envoy:v1.16.5"
  description = "Using the debian base images as alpine does not support arm"
}

//variable "consul_image" {
//  default = "hashicorp/consul:1.11.1"
//}
//
//variable "consul_envoy_image" {
//  default     = "envoyproxy/envoy:v1.20.0"
//  description = "Using the debian base images as alpine does not support arm"
//}

variable "consul_tls_enabled" {
  description = "Enable TLS to secure the Consul server"
  default     = true
}

variable "consul_debug" {
  description = "Log debug mode"
  default     = true
}

variable "controller_enabled" {
  description = "Should the controller be installed with the helm chart?"
  default     = true
}

network "dc1" {
  subnet = "10.5.0.0/16"
}

k8s_cluster "dc1" {
  driver = "k3s"

  nodes = 1

  network {
    name = "network.dc1"
  }
}

// install cert manager
k8s_config "cert-manager-controller" {
  cluster = "k8s_cluster.dc1"

  paths = [
    "${file_dir()}/cert-manager.yaml",
  ]

  wait_until_ready = true

  health_check {
    timeout = "60s"
    pods    = ["app.kubernetes.io/instance=cert-manager"]
  }
}

module "consul" {
  source = "github.com/shipyard-run/blueprints?ref=8756e198e5b402ae93f23320c75b9ff519d38c2c/modules//kubernetes-consul"
}

ingress "web" {
  source {
    driver = "local"

    config {
      port = 9092
    }
  }

  destination {
    driver = "k8s"

    config {
      cluster = "k8s_cluster.dc1"
      address = "web.default.svc"
      port    = 9090
    }
  }
}

k8s_config "application" {
  depends_on = [
    "module.consul",
  ]

  cluster = "k8s_cluster.dc1"

  paths = [
    "${file_dir()}/../../example/kubernetes/",
  ]

  wait_until_ready = true
}

output "KUBECONFIG" {
  value = k8s_config("dc1")
}
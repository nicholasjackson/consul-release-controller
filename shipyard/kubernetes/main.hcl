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

variable "consul_release_controller_enabled" {
  description = "Enable the Consul release controller using the blueprint"
  default     = false
}

//variable "consul_image" {
//  default = "hashicorp/consul:1.9.13"
//}
//
//variable "consul_envoy_image" {
//  default     = "envoyproxy/envoy:v1.16.5"
//  description = "Using the debian base images as alpine does not support arm"
//}

variable "consul_image" {
  default = "hashicorp/consul:1.11.5"
}
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

variable "example_app" {
  description = "Should the example application be installed?"
  default     = true
}

variable "helm_chart_install" {
  description = "Should the Helm chart for the controller be installed?"
  default     = true
}

variable "helm_controller_enabled" {
  description = "Should the controller be enabled in the Helm chart? You may want to set this to false if using a local controller"
  default     = true
}

variable "controller_repo" {
  description = "Docker repo for the controller"
  default     = "nicholasjackson/consul-release-controller"
}

variable "controller_version" {
  description = "Docker image version for the controller"
  default     = ""
}

variable "controller_image" {
  default = "${var.controller_version != "" ? "${var.controller_repo}:${var.controller_version}" : ""}"
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

  image {
    name = var.controller_image
  }
}

module "consul" {
  source = "github.com/shipyard-run/blueprints?ref=d0ef4e69081263dad65edcf05ea3dc8da395b418/modules//kubernetes-consul"
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
  disabled = !var.example_app

  depends_on = [
    "module.consul",
  ]

  cluster = "k8s_cluster.dc1"

  paths = [
    "${file_dir()}/../../example/kubernetes/basic"
  ]

  wait_until_ready = true
}

output "KUBECONFIG" {
  value = k8s_config("dc1")
}
variable "example_app" {
  description = "Should the example application be installed?"
  default     = false
}

variable "helm_chart_install" {
  description = "Should the Helm chart for the controller be installed?"
  default     = false
}

module "dev_env" {
  source = "github.com/nicholasjackson/consul-release-controller//shipyard/kubernetes"
}
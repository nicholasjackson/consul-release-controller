variable "example_app" {
  description = "Should the example application be installed?"
  default     = true
}

variable "helm_chart_install" {
  description = "Should the Helm chart for the controller be installed?"
  default     = true
}

module "dev_env" {
  source = "../kubernetes"
}
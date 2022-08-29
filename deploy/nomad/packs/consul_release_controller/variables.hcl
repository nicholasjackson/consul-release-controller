variable "controller_image" {
  description = "The name of the Consul Release Controller image to deploy"
  type        = string
  default     = "nicholasjackson/consul-release-controller:3c108be.dev"
}

variable "tls_cert" {
  description = "The TLS certificate to use to secure the API transport"
  type        = string
  default     = ""
}

variable "tls_key" {
  description = "The TLS key to use to secure the API transport"
  type        = string
  default     = ""
}

variable "nomad_addr" {
  default = "http://$${attr.unique.network.ip-address}:4646"
}

variable "consul_http_addr" {
  default = "http://$${attr.unique.network.ip-address}:8500"
}
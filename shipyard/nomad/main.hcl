// set the variable for the network
variable "cn_network" {
  default = "dc1"
}

variable "cn_nomad_cluster_name" {
  default = "nomad_cluster.local"
}

variable "cn_nomad_client_nodes" {
  default = 0
}

network "dc1" {
  subnet = "10.5.0.0/16"
}

variable "cn_nomad_client_config" {
  default = "${data("nomad_config")}/client.hcl"
}

# Create a nomad host volume that waypoint can write persistent data to
variable "cn_nomad_client_host_volume" {
  default = {
    name        = "waypoint"
    source      = data("waypoint")
    destination = "/data"
    type        = "bind"
  }
}

variable "install_monitoring" {
  default = true
}

variable "install_waypoint" {
  default = true
}

variable "install_vault" {
  default = true
}

variable "install_controller" {
  default = "docker"
  #default = "local"
}

variable "install_example_app" {
  default = true
}

module "consul_nomad" {
  depends_on = ["container.waypoint-odr"]
  source     = "github.com/shipyard-run/blueprints?ref=06822657c974816597dacecad6ee3a90af6809e3/modules//consul-nomad"
  #source = "/home/nicj/go/src/github.com/shipyard-run/blueprints/modules/consul-nomad"
}

module "monitoring" {
  depends_on = ["module.consul_nomad"]
  disabled   = !var.install_monitoring

  source = "./modules/monitoring"
}

module "waypoint" {
  disabled = !var.install_waypoint

  source = "./modules/waypoint"
}

module "example_app" {
  disabled = !var.install_example_app

  source = "./modules/example_app"
}

module "controller" {
  depends_on = ["module.consul_nomad"]
  disabled   = var.install_controller == ""

  source = "./modules/releaser"
}

module "vault" {
  disabled = !var.install_vault

  source = "./modules/vault"
}

#module "boundary" {
#  disabled = var.install_controller == ""
#
#  source = "./modules/releaser"
#}
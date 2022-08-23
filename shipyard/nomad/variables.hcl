variable "cn_network" {
  default = "dc1"
}

variable "pack_folder" {
  default = "${file_dir()}/../../deploy/nomad/packs"
}

variable "cn_nomad_cluster_name" {
  default = "nomad_cluster.local"
}

variable "cn_nomad_client_nodes" {
  default = 1
}

variable "cn_nomad_version" {
  default = "1.3.1"
}

variable "cn_nomad_client_config" {
  default = "${data("nomad_config")}/client.hcl"
}

# Set these variables to false to disable a particular module
variable "install_nomad" {
  default = true
}

variable "install_monitoring" {
  default = true
}

variable "install_controller" {
  default = "docker"
  #default = "local"
}

variable "install_example_app" {
  default = true
}

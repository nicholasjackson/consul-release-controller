variable "controller_image" {
  default = "docker.io/nicholasjackson/consul-release-controller:3c108be.dev"
}

template "controller_pack" {
  depends_on = ["exec_local.generate_certs"]
  disabled   = var.install_controller != "docker"

  source = <<-EOF
    #!/bin/sh
    cat <<-EOH > /scripts/release_controller.hcl
    tls_cert = <<-EOT
    $(cat /certs/leaf.cert)
    EOT
    
    tls_key = <<-EOT
    $(cat /certs/leaf.key)
    EOT
    EOH

    nomad-pack run \
      -f /scripts/release_controller.hcl \
      /pack/nomad-pack-community-registry-main/packs/consul_release_controller
  EOF

  destination = "${data("controller_data")}/install_release_controller.sh"
}

exec_remote "controller_pack" {
  depends_on = ["nomad_cluster.local", "template.controller_pack"]
  disabled   = var.install_controller != "docker"

  image {
    name = "shipyardrun/hashicorp-tools:v0.8.0"
  }

  network {
    name = "network.dc1"
  }

  cmd = "/bin/bash"
  args = [
    "/scripts/install_release_controller.sh"
  ]

  # Mount a volume containing the config
  volume {
    source      = "${file_dir()}/../../pack"
    destination = "/pack"
  }

  volume {
    source      = data("nomad_data")
    destination = "/certs"
  }

  volume {
    source      = data("controller_data")
    destination = "/scripts"
  }

  working_directory = "/pack"

  env {
    key   = "NOMAD_ADDR"
    value = "http://server.local.nomad-cluster.shipyard.run:4646"
  }
}

nomad_ingress "controller-docker" {
  disabled = var.install_controller != "docker"

  cluster = var.cn_nomad_cluster_name
  job     = "release-controller"
  group   = "release-controller"
  task    = "release-controller"

  network {
    name = "network.dc1"
  }

  port {
    local  = 9443
    remote = "server"
    host   = 9443
  }
}

output "consul_release_controller_addr" {
  value = "https://localhost:9443"
}
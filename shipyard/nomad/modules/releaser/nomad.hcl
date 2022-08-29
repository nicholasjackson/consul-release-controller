variable "controller_image" {
  default = "docker.io/nicholasjackson/consul-release-controller:ca90d3b.dev"
}

template "controller_pack" {
  depends_on = ["certificate_leaf.releaser_leaf"]
  disabled   = var.install_controller != "docker"

  source = <<-EOF
    #!/bin/sh

    cat <<-EOH > /scripts/release_controller.hcl
    controller_image = "${var.controller_image}"

    tls_cert = <<-EOT
    $(cat /certs/releaser_leaf.cert)
    EOT
    
    tls_key = <<-EOT
    $(cat /certs/releaser_leaf.key)
    EOT
    EOH

    nomad-pack run \
      -f /scripts/release_controller.hcl \
      /pack/consul_release_controller
  EOF

  destination = "${data("controller_data")}/install_release_controller.sh"
}

exec_remote "controller_pack" {
  depends_on = ["nomad_cluster.local", "template.controller_pack"]
  disabled   = var.install_controller != "docker"

  image {
    name = "shipyardrun/hashicorp-tools:v0.9.0"
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
    source      = var.pack_folder
    destination = "/pack"
  }

  volume {
    source      = data("nomad_config")
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

output "consul_release_controller_addr" {
  value = "http://releases.ingress.shipyard.run"
}

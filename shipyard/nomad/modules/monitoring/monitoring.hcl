template "monitoring_pack" {
  source = <<-EOF
    #!/bin/bash -e

    nomad-pack run -f /pack/prometheus-pack.hcl /pack/nomad-pack-community-registry-main/packs/prometheus
    nomad-pack run -f /pack/grafana.hcl /pack/nomad-pack-community-registry-main/packs/grafana
    nomad-pack run -f /pack/loki.hcl /pack/nomad-pack-community-registry-main/packs/loki
  EOF

  destination = "${data("monitoring_data")}/install_monitoring.sh"
}

exec_remote "pack" {
  depends_on = ["nomad_cluster.local", "template.monitoring_pack"]

  image {
    name = "shipyardrun/hashicorp-tools:v0.8.0"
  }

  network {
    name = "network.dc1"
  }

  cmd = "/bin/bash"
  args = [
    "/scripts/install_monitoring.sh"
  ]

  # Mount a volume containing the config
  volume {
    source      = "${file_dir()}/../../pack"
    destination = "/pack"
  }

  volume {
    source      = data("monitoring_data")
    destination = "/scripts"
  }

  working_directory = "/pack"

  env {
    key   = "NOMAD_ADDR"
    value = "http://server.local.nomad-cluster.shipyard.run:4646"
  }
}

nomad_ingress "grafana" {
  cluster = var.cn_nomad_cluster_name
  job     = "grafana"
  group   = "grafana"
  task    = "grafana"

  network {
    name = "network.dc1"
  }

  port {
    local  = 3000
    remote = "http"
    host   = 3000
  }
}

nomad_ingress "prometheus" {
  cluster = var.cn_nomad_cluster_name
  job     = "prometheus"
  group   = "prometheus"
  task    = "prometheus"

  network {
    name = "network.dc1"
  }

  port {
    local  = 9090
    remote = "http"
    host   = 9090
  }
}
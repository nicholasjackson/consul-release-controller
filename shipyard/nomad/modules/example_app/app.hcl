exec_remote "proxy-defaults" {
  depends_on = ["container.consul"]

  image {
    name = "consul:1.12.0"
  }

  network {
    name = "network.dc1"
  }

  cmd = "/bin/sh"
  args = [
    "set_defaults.sh"
  ]

  # Mount a volume containing the config
  volume {
    source      = "${file_dir()}/consul_config"
    destination = "/config"
  }

  working_directory = "/config"

  env {
    key   = "CONSUL_HTTP_ADDR"
    value = "http://consul.container.shipyard.run:8500"
  }
}

nomad_job "jobs" {
  cluster = var.cn_nomad_cluster_name
  paths = [
    "${file_dir()}/jobs/loadtest.hcl",
    "${file_dir()}/jobs/ingress.hcl",
    "${file_dir()}/jobs/api.hcl",
    "${file_dir()}/jobs/payments.hcl"
  ]
}

nomad_ingress "ingress" {
  cluster = var.cn_nomad_cluster_name
  job     = "ingress"
  group   = "ingress"
  task    = "ingress"

  network {
    name = "network.dc1"
  }

  port {
    local  = 18080
    remote = "inbound"
    host   = 18081
  }
}

nomad_ingress "api" {
  cluster = var.cn_nomad_cluster_name
  job     = "api-deployment"
  group   = "api"
  task    = "api"

  network {
    name = "network.dc1"
  }

  port {
    local  = 9090
    remote = "http"
    host   = 19090
  }
}
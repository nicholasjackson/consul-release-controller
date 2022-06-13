exec_local "generate_certs" {
  cmd = "shipyard"
  args = [
    "connector",
    "generate-certs",
    "--leaf",
    "--root-ca",
    "./root.cert",
    "--root-key",
    "./root.key",
    "--dns-name",
    "127.0.0.1:9443",
    "${data("nomad_data")}",
  ]

  working_directory = "${shipyard()}/certs"

  timeout = "30s"
}

output "TLS_CERT" {
  value = "${data("nomad_data")}/leaf.cert"
}

output "TLS_KEY" {
  value = "${data("nomad_data")}/leaf.key"
}

nomad_job "controller-local" {
  disabled = var.install_controller != "local"

  cluster = var.cn_nomad_cluster_name
  paths = [
    "./jobs/controller.hcl",
  ]
}

nomad_ingress "controller-local" {
  disabled = var.install_controller != "local"

  cluster = var.cn_nomad_cluster_name
  job     = "release-controller"
  group   = "release-controller"
  task    = "socat"

  network {
    name = "network.dc1"
  }

  port {
    local  = 8080
    remote = "http"
    host   = 18080
  }
}
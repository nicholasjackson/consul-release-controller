ingress "upstreams-proxy" {
  source {
    driver = "local"

    config {
      port = 28080
    }
  }

  destination {
    driver = "k8s"

    config {
      cluster = "k8s_cluster.dc1"
      address = "consul-release-controller.default.svc"
      port    = 8080
    }
  }
}

k8s_config "upstreams-proxy" {
  depends_on = ["module.consul"]

  cluster = "k8s_cluster.dc1"
  paths = [
    "./fake-controller.yaml",
  ]

  wait_until_ready = true
}

output "UPSTREAMS" {
  value = "http://localhost:28080"
}
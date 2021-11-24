network "dc1" {
  subnet = "10.5.0.0/16"
}

k8s_cluster "dc1" {
  driver = "k3s"

  nodes = 1

  network {
    name = "network.dc1"
  }
}

// install cert manager
k8s_config "cert-manager" {
  cluster = "k8s_cluster.dc1"

  paths = [
    "${file_dir()}/cert-manager.yaml",
  ]

  wait_until_ready = true

  health_check {
    timeout = "60s"
    pods    = ["app.kubernetes.io/instance=cert-manager"]
  }
}

output "KUBECONFIG" {
  value = k8s_config("dc1")
}

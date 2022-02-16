template "controller_values" {
  source = <<EOF
controller:
  enabled: "${var.controller_enabled}"
  container_config:
    image:
      repository: "${var.controller_repo}"
      tag: "${var.controller_version}"
autoencrypt:
  enabled: ${var.consul_tls_enabled}
acls:
  enabled: ${var.consul_acls_enabled}
  EOF

  destination = "${data("kube_setup")}/helm-values.yaml"
}

helm "consul-release-controller" {

  # wait for certmanager to be installed and the template to be processed
  depends_on = [
    "template.controller_values",
    "k8s_config.cert-manager-controller",
    "module.consul",
  ]

  cluster          = "k8s_cluster.dc1"
  namespace        = "consul"
  create_namespace = true

  chart = "../../deploy/kubernetes/charts/consul-release-controller"

  values = "${data("kube_setup")}/helm-values.yaml"
}

// fetch the certifcates
template "certs_script" {
  source = <<EOF
#! /bin/sh -e

kubectl get secret consul-release-controller-webhook-certificate -n consul -o json | \
	jq -r '.data."tls.crt"' | \
	base64 -d > /output/tls.crt

kubectl get secret consul-release-controller-webhook-certificate -n consul -o json | \
	jq -r '.data."tls.key"' | \
	base64 -d > /output/tls.key
  EOF

  destination = "${data("kube_setup")}/fetch_certs.sh"
}

exec_remote "exec_standalone" {
  depends_on = [
    "helm.consul-release-controller",
    "template.certs_script",
  ]

  network {
    name = "network.dc1"
  }

  image {
    name = "shipyardrun/tools:v0.6.0"
  }

  cmd = "sh"
  args = [
    "/output/fetch_certs.sh",
  ]

  volume {
    source      = "${data("kube_setup")}"
    destination = "/output"
  }

  volume {
    source      = k8s_config_docker("dc1")
    destination = "/kubeconfig.yaml"
  }

  env {
    key   = "KUBECONFIG"
    value = "/kubeconfig.yaml"
  }
}

output "TLS_CERT" {
  value = "${data("kube_setup")}/tls.crt"
}

output "TLS_KEY" {
  value = "${data("kube_setup")}/tls.key"
}

ingress "controller-webhook" {
  source {
    driver = "k8s"

    config {
      cluster = "k8s_cluster.dc1"
      port    = 9443
    }
  }

  destination {
    driver = "local"

    config {
      address = "localhost"
      port    = 9443
    }
  }
}
template "controller_values" {
  disabled = !var.helm_chart_install

  source = <<EOF
controller:
  enabled: "#{{ .Vars.controller_enabled }}"
  container_config:
    image:
      repository: "#{{ .Vars.controller_repo }}"
      tag: "#{{ .Vars.controller_version }}"
autoencrypt:
  enabled: #{{ .Vars.tls_enabled }}
acls:
  enabled: #{{ .Vars.acls_enabled }}
#{{- if eq .Vars.controller_enabled false }}
webhook:
  additionalDNSNames:
    - controller-webhook.shipyard.svc
  serviceOverride: controller-webhook
  namespaceOverride: shipyard
#{{ end }}
EOF

  destination = "${data("kube_setup")}/helm-values.yaml"

  vars = {
    controller_enabled = var.helm_controller_enabled
    acls_enabled       = var.consul_acls_enabled
    tls_enabled        = var.consul_tls_enabled
    controller_repo    = var.controller_repo
    controller_version = var.controller_version
  }
}

//template "controller_service" {
//  disabled = !var.helm_chart_install
//
//  source = <<EOF
//---
//apiVersion: v1
//kind: Service
//metadata:
//  name: external-webhook
//  namespace: consul
//spec:
//  type: ExternalName
//  externalName: controller-webhook.shipyard.svc
//  EOF
//
//  destination = "${data("kube_setup")}/service.yaml"
//}
//
//k8s_config "controller_service" {
//  disabled = !var.helm_chart_install
//
//  depends_on = [
//    "module.consul",
//  ]
//
//  cluster = "k8s_cluster.dc1"
//
//  paths = [
//    "${data("kube_setup")}/service.yaml"
//  ]
//
//  wait_until_ready = true
//}

helm "consul-release-controller" {
  disabled = !var.helm_chart_install

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
  disabled = !var.helm_chart_install

  source = <<EOF
#! /bin/sh -e

kubectl get secret consul-release-controller-certificate -n consul -o json | \
	jq -r '.data."tls.crt"' | \
	base64 -d > /output/tls.crt

kubectl get secret consul-release-controller-certificate -n consul -o json | \
	jq -r '.data."tls.key"' | \
	base64 -d > /output/tls.key
  EOF

  destination = "${data("kube_setup")}/fetch_certs.sh"
}

exec_remote "exec_standalone" {
  disabled = !var.helm_chart_install

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

ingress "controller-webhook" {
  disabled = !var.helm_chart_install

  source {
    driver = "k8s"

    config {
      cluster = "k8s_cluster.dc1"
      port    = 19443
    }
  }

  destination {
    driver = "local"

    config {
      address = "localhost"
      port    = 19443
    }
  }
}

output "TLS_CERT" {
  disabled = !var.helm_chart_install

  value = "${data("kube_setup")}/tls.crt"
}

output "TLS_KEY" {
  disabled = !var.helm_chart_install

  value = "${data("kube_setup")}/tls.key"
}

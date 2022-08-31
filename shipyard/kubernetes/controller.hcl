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
#{{- if eq .Vars.acls_enabled true }}
acls:
  secretName: "consul-controller-token"
  secretKey: "token"
#{{ end }}
#{{- if eq .Vars.controller_enabled false }}
webhook:
  service: controller-webhook
  namespace: shipyard
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

template "cert_manager_values" {
  disabled = !var.helm_chart_install

  source = <<-EOF
  installCRDs: 'true'
  # Disable Transparent proxy and Consul injection
  startupapicheck:
    podAnnotations:
      'consul.hashicorp.com/connect-inject': 'false'
      'consul.hashicorp.com/transparent-proxy': 'false'
  cainjector:
    podAnnotations:
      'consul.hashicorp.com/connect-inject': 'false'
      'consul.hashicorp.com/transparent-proxy': 'false'
  webhook:
    podAnnotations:
      'consul.hashicorp.com/connect-inject': 'false'
      'consul.hashicorp.com/transparent-proxy': 'false'
  podAnnotations:
    'consul.hashicorp.com/connect-inject': 'false'
    'consul.hashicorp.com/transparent-proxy': 'false'
  EOF

  destination = "${data("kube_setup")}/cert-manager-helm-values.yaml"
}

template "controller_acl_token" {
  disabled   = !var.helm_chart_install
  depends_on = ["module.consul"]

  source = <<-EOF
  kubectl create secret generic consul-controller-token \
    --from-literal='token=#{{ file .Vars.token_file | trim }}' \
    -n consul
  EOF

  destination = "${data("kube_setup")}/controller-acl-token.sh"

  vars = {
    token_file = var.controller_token_file
  }
}

exec_remote "bootstrap_controller_acls" {
  disabled = !var.helm_chart_install

  depends_on = ["template.controller_acl_token", "module.consul"]

  image {
    name = "shipyardrun/tools:v0.6.0"
  }

  network {
    name = "network.${var.consul_k8s_network}"
  }

  cmd = "sh"
  args = [
    "/data/controller-acl-token.sh",
  ]

  # Mount a volume containing the config for the kube cluster
  volume {
    source      = "${shipyard()}/config/${var.consul_k8s_cluster}"
    destination = "/config"
  }

  volume {
    source      = data("kube_setup")
    destination = "/data"
  }

  env {
    key   = "KUBECONFIG"
    value = "/config/kubeconfig-docker.yaml"
  }
}

helm "cert_manager_local" {
  disabled = !var.helm_chart_install

  cluster          = "k8s_cluster.dc1"
  namespace        = "cert-manager"
  create_namespace = true

  repository {
    name = "jetstack"
    url  = "https://charts.jetstack.io"
  }

  chart   = "jetstack/cert-manager"
  version = "v1.8.0"

  health_check {
    timeout = "120s"
    pods = [
      "app=cert-manager",
    ]
  }

  values = "${data("kube_setup")}/cert-manager-helm-values.yaml"
}

helm "consul_release_controller" {
  disabled = !var.helm_chart_install

  # wait for Consul to be installed and the template to be processed
  depends_on = [
    "template.controller_values",
    "helm.cert_manager",
    "module.consul",
  ]

  cluster          = "k8s_cluster.dc1"
  namespace        = "consul"
  create_namespace = true

  chart = "../../deploy/kubernetes/charts/consul-release-controller"

  values = "${data("kube_setup")}/helm-values.yaml"
}

// fetch the certifcates
template "get_certs" {
  disabled = !var.helm_chart_install

  source = <<EOF
#!/bin/bash

getSecret () {
  kubectl get secret consul-release-controller-certificate -n consul -o json | \
    jq -r ".data.\"$1\"" | \
    base64 -d > /output/$1

  [ -s /output/$1 ] || return 1

  return 0
}

max_retry=5
counter=0
until getSecret "tls.crt"
do
  sleep 10
  [[ $counter -eq $max_retry ]] && echo "Failed!" && exit 1
  echo "Trying again. Try #$counter"
  ((counter++))
done

counter=0
until getSecret "tls.key"
do
  sleep 10
  [[ $counter -eq $max_retry ]] && echo "Failed!" && exit 1
  echo "Trying again. Try #$counter"
  ((counter++))
done
  EOF

  destination = "${data("kube_setup")}/fetch_certs.sh"
}

exec_remote "get_certs" {
  disabled = !var.helm_chart_install

  depends_on = [
    "helm.consul_release_controller",
    "template.get_certs",
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
    source      = data("kube_setup")
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

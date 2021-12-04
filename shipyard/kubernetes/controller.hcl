template "controller_values" {
  source = <<EOF
controller:
  enabled: "false"

webhook:
  type: ClusterIP
  port: 9443
  service: controller-webhook
  namespaceOverride: shipyard
  failurePolicy: Ignore

  # Allows adding additional DNS Names to the cert generated
  # for the webhook
  additionalDNSNames:
    - "controller-webhook.shipyard.svc"

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

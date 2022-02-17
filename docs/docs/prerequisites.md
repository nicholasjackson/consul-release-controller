---
sidebar_position: 4
---

# Prerequisites

## Kubernetes
Consul Release Controller has been tested on Kubernetes version 1.21+, it may work on other versions but we 
only plan on supporting the three most recent [Kubernetes releases](https://kubernetes.io/releases/).

## Cert Manager
The Consul Release Controller secures its API and Kubernetes webhook using TLS, [cert-manager](https://cert-manager.io/docs/)
is used to create these self signed certificates.

```yaml
---
# Source: consul-release-controller/templates/controller-cert.yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: consul-release-controller-certificate
  namespace: "default"
spec:
  secretName: consul-release-controller-certificate
  dnsNames:
  - "consul-release-controller-webhook.default.svc"
  issuerRef:
    name: controller-consul-release-controller-selfsigned
---
# Source: consul-release-controller/templates/controller-cert.yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: controller-consul-release-controller-selfsigned
  namespace: "default"
spec:
  selfSigned: {}
```

Please see the [cert-manager installation guide](https://cert-manager.io/docs/installation/) for details on how to setup 
cert manager for your cluster.

## HashiCorp Consul
Consul Release Controller uses Consul Service Mesh (Connect) for the provision of metrics that are used to determine 
application health and for [routing features](https://www.consul.io/docs/connect/config-entries) that enable 
precise control over the traffic for your service.

Consul Release Controller requires Consul version 1.9+, but only requires the ability to interact with the Consul 
API. It does not matter if you have a installed Consul locally on your Kubernetes cluster using [Helm](https://www.consul.io/docs/k8s/helm), 
[HCP Consul](https://cloud.hashicorp.com/), or if you are using an external server.

## Prometheus
To understand the health of your application Consul Release Controller reads the metrics scraped from the Envoy proxy in Consul service
mesh. At present the only supported time series database is [Prometheus](https://prometheus.io/).  Like Consul, the Release
Controller only needs to be able to access the Prometheus API, it should not matter if you are using [Grafana Cloud](https://grafana.com/products/cloud/), the [Prometheus operator](https://github.com/prometheus-operator/prometheus-operator).

## Grafana
Grafana is not directly required for Consul Release Controller to function however it is incredibly useful for visualizing the 
metrics from Prometheus.
# Consul Release Controller

Consul release controller enables automated Canary and Green / Blue releases of your application using Consul Service Mesh.

## Documentation

For information on how to install the release controller and full documentation [click here](https://nicholasjackson.io/consul-release-controller/)

## Why not Flagger / Argo / etc. ?
Consul Release Controller is designed to be cloud and platform agnostic, while it is a great tool to use with Kubernetes it allows a common
workflow on any platform supported by Consul such as Nomad, ECS, or Virtual machines. Since Consul Release Controller is specific to consul
service mesh it can leverage Consul full capabilities such as cross cloud deployments, or multi-platform canary releases (e.g. migrating applications
from on premise VMs to cloud based Nomad or Kubernetes). 

## Supported platforms [x] currently supported:

- [x] Kubernetes
- [x] Nomad
- [ ] ECS
- [ ] Virtual machines
- [ ] Cross platform deployments (e.g. ECS to Kubernetes, Virtual Machine to Nomad) 
- [x] Enterprise feature support


# Configuration

The controller can be configured by setting the following environment variables:

### TLS_CERT
Path to the TLS certificate file for securing the main restful API.

### TLS_KEY
Path to the TLS key file for securing the main restful API.

### KUBECONFIG
Path to the kubeconfig file for connecting to the Kubernetes cluster.

### CONSUL_HTTP_ADDR
Address of the Consul cluster.

### CONSUL_HTTP_TOKEN_FILE
Path to the Consul token file containing a valid ACL token for the Consul cluster

### CONSUL_CA_CERT
Path to the CA certificate file for making requests to the Consul cluster.

## Developing

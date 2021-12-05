# Consul Release Controller

Consul release controller enables automated Canary and Green / Blue releases of your application using Consul Service Mesh.

## Why not Flagger / Argo / etc. ?
Consul Release Controller is designed to be cloud and platform agnostic, while it is a great tool to use with Kubernetes it allows a common
workflow on any platform supported by Consul such as Nomad, ECS, or Virtual machines. Since Consul Release Controller is specific to consul
service mesh it can leverage Consul full capabilities such as cross cloud deployments, or multi-platform canary releases (e.g. migrating applications
from on premise VMs to cloud based Nomad or Kubernetes). 

## Supported platforms [x] currently supported:
[x] Kubernetes
[ ] Nomad
[ ] ECS
[ ] Virtual machines
[ ] Cross platform deployments (e.g. ECS to Kubernetes, Virtual Machine to Nomad) 
[ ] Enterprise feature support

## Developing

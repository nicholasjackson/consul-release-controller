---
sidebar_position: 1
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# Getting Started on Kubernetes

To install the Consul release controller you need the following prerequisites:

* Kubernetes
* Cert Manager *(for creating the TLS certificate to enable SSL on the controller)*
* Consul with the Connect feature enabled *(Consul Release Controller relies on Consul Service Mesh)*
* Prometheus / Grafana *(for querying service health and displaying dashboards)*

For more information on the prequisites, please see the [prerequisites documentation](prerequisites).

This guide assumes that you have a cluster with the pre-requisites installed.  You can also find details on how to setup a `dev` environment on your local machine using `Shipyard` and `Docker`

## Installing Consul Release Controller

Consul release controller can be installed to your Kubernetes cluster using the Helm chart, you can add the Helm chart repo using the following
command.

```shell
helm repo add release-controller https://nicholasjackson.io/consul-release-controller/
```

To test that the repo has been correctly added you can use the following command.

```shell
helm search repo release-controller --versions
```

You should see the available versions output

```shell
NAME                                            CHART VERSION   APP VERSION
release-controller/consul-release-controller    0.0.6           0.0.6      
```

The Helm chart is configurable with a number of settings however this guide assumes that you have installed Consul locally on your 
Kubernetes cluster using the recommended secure settings with TLS and ACLs enabled.

<details>
  <summary>Official Consul Helm chart <b>values.yaml</b></summary>

```yaml
global:
  acls:
    manageSystemACLs: true
controller:
  enabled: true
acls:
  enabled: true
```
</details>

To install using the command to install the release controller:

<Tabs groupId="helm_values">
<TabItem value="secure" label="TLS and ALCs">


```shell
helm install consul-release-controller \
  release-controller/consul-release-controller \
  -n consul \
  --set "autoEncrypt.enabled=true" \
  --set "acls.enabled=true"
```

</TabItem>
<TabItem value="insecure" label="Insecure setup">

```shell
helm install consul-release-controller \
  release-controller/consul-release-controller
```

</TabItem>
</Tabs>

The controller will be installed into the namespace `consul` with the default options setting the agent certificate and the ACL token from the Consul Kubernetes controller

**note:** The release controller shares secrets with the main Consul install, ensure that you install it into the same namespace where the 
Consul helm is installed.

```shell
NAME: consul-release-controller
LAST DEPLOYED: Tue Feb 15 17:12:31 2022
NAMESPACE: default
STATUS: deployed
REVISION: 1
```

For custom configuration please see all the configurable options in the [Helm Values](helm_values)

You can validate that the controller has been successfully installed and is running by querying the Kubernetes pods, you should see a single
container for the release controller.

```shell
kubectl get deployments consul-release-controller -n consul
```

```shell
NAME                        READY   UP-TO-DATE   AVAILABLE   AGE
consul-release-controller   1/1     1            1           53s
```
---
sidebar_position: 1
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# Getting Started on Kubernetes

To install the Consul release controller you need the following pre-requisites:

* A Kubernetes Cluster
* Cert Manager
* Consul with the Connect feature enabled
* Prometheus / Grafana

This guide assumes that you have a cluster with the pre-requisites installed, help with setting up `dev` versions of the pre-requisites can
be found the bottom of this guide. You can also find details on how to setup a `dev` environment on your local machine using `Shipyard` and `Docker`

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
release-controller/consul-release-controller    0.0.4           0.0.4      
```

### Helm values

Depending on the security configuration of your Consul server, you need to configure the Helm values file accordingly, Consul Release Controller
needs to communicate to a Consul agent in order to set the various Service Mesh Configuration.

Consul release controller can be configured using the same environment variables used for Consul Agent.

[https://www.consul.io/commands#environment-variables](https://www.consul.io/commands#environment-variables)


Depending on your security configuration you will need to configure the Helm chart to set these values and to obtain the 
associated certificates or tokens.

<Tabs groupId="helm_values">
<TabItem value="insecure" label="Insecure">

If you are using a Consul setup that does not have ACLs configured or TLS security enabled, the default values
assume that the Consul Agent for the server is installed as a `DaemonSet`, and is available using the Kubernetes
`status.hostIP`

```yaml
controller:

  container_config:
    - name: HOST_IP
      valueFrom:
        fieldRef:
          fieldPath: status.hostIP
    - name: CONSUL_HTTP_ADDR
      value: https://$(HOST_IP):8501
```

If your cluster is not setup in this way then you will need to change the environment variable `CONSUL_HTTP_ADDR` to the address
of a Consul Agent that can be used by the cluster. Consul release controller will work if pointed directly at the Consul server,
however this is not recommend. 

While fine for local development environments, we do not recommend using Consul without ACL's and 
TLS.

</TabItem>
  
<TabItem value="secure" label="ACLS and TLS" default="true">

If your Consul server is configured to use ACLs and TLS you will need to modify the Helm values to add the CA certificate that 
the Agent is using to expose the API and also the ACL Token with the correct permissions to write Consul config.

These can be exposed through environment variables as shown in the snippets below. If you are using a the Consul CRDs controller for Kubernetes, you 
can simply use the same token for the Consul release controller as both require the same permissions to read and write Consul config.

The ACL token can be obtained from the `consul-controller-acl-token` secret that is created when installing the CRDs controller from the main Consul
Helm chart.

#### Environment variables

```yaml
---
controller:

  container_config:
    env:
    - name: HOST_IP
      valueFrom:
        fieldRef:
          fieldPath: status.hostIP
    - name: CONSUL_HTTP_ADDR
      value: https://$(HOST_IP):8501
    - name: CONSUL_CAPATH
      value: /consul/tls/client/ca/tls.crt
    - name: CONSUL_HTTP_TOKEN
      valueFrom: 
        secretKeyRef:
          name: consul-controller-acl-token
          key: token 
    
    additional_volume_mounts:
      - mountPath: /consul/tls/client/ca
        name: consul-auto-encrypt-ca-cert
```

#### CA certificate

To validate the authenticity of the server, you will need the root certificate that the agent is using. When using Consul`s autoencrypt feature
the root certificate that the agent uses is different from the one that the server uses. Autoencypt uses the connect certificate authority. To
get the certificate you can either.

First you need to add volumes that can be shared between the init container and the controller, when auto encrypt is enabled in Consul, the agent
and the server both use different CAs to secure the API. To fetch the client CA you can use the `consul-k8s` command, however you will first need
the server CA in order to make a secure request to the server to request the client ca. 

The server ca you can be obtained from the secret `consul-server-ca`, this is mounted as a volume so that the `consul-k8s` command can read it.

The second volume `consul-auto-encrypt-ca-cert` is set as an in memory volume, the `consul-k8s` command will write the client ca to this location.

```yaml
controller:

  additional_volumes: 
    - name: consul-server-ca
      secret:
        secretName: consul-server-cert
    - name: consul-auto-encrypt-ca-cert
      emptyDir:
        medium: Memory
```

To fetch the client CA you can use the `consul-k8s` command that automates the process of fetching the certificate
from the Consul server. This process can be run in an init container alongside the release controller container, the following YAML snippet
shows how you can use `consul-k8s` to fetch the certificate and write it to the shared memory volume `consul-auto-encrypt-ca-cert`.

```yaml
controller:

  additional_init_containers:
    - command:
      - /bin/sh
      - -ec
      - |
        consul-k8s get-consul-client-ca \
          -output-file=/consul/tls/client/ca/tls.crt \
          -server-addr=consul-server \
          -server-port=8501 \
          -ca-file=/consul/tls/ca/tls.crt
      image: hashicorp/consul-k8s:0.25.0
      imagePullPolicy: IfNotPresent
      name: get-auto-encrypt-client-ca
      resources:
        limits:
          cpu: 50m
          memory: 50Mi
        requests:
          cpu: 50m
          memory: 50Mi
      volumeMounts:
      - mountPath: /consul/tls/ca
        name: consul-server-ca
      - mountPath: /consul/tls/client/ca
        name: consul-auto-encrypt-ca-cert
```

</TabItem>
</Tabs>
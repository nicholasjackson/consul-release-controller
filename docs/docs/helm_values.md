---
sidebar_position: 5
---

# Helm values

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
      value: http://$(HOST_IP):8501
```

If your cluster is not setup in this way then you will need to change the environment variable `CONSUL_HTTP_ADDR` to the address
of a Consul Agent that can be used by the cluster. Consul release controller will work if pointed directly at the Consul server,
however this is not recommend. 

While fine for local development environments, we do not recommend using Consul without ACL's and TLS.

</TabItem>
  
<TabItem value="secure" label="ACLS and TLS" default="true">

#### ACL support
If you have setup Consul using the official Helm chart and have enabled ACL and the Kubernetes controller using the following Helm values:

```yaml title="Official Consul Helm Values"
global:
  acls:
    manageSystemACLs: true
controller:
  enabled: true
```

You can simply enable ACL support in the Helm values:

```yaml title="Consul Release Controller Helm Values"
acls:
  enabled: true
```

The Helm chart will auto configure the controller to set the environment variable `CONSUL_HTTP_TOKEN` to use the same ACL token stored in the secret
`consul-controller-acl-token` as used by the Kubernetes controller.  Consul Release Controller and the Kubernetes Controller both require the same 
permissions to read and write Consul config.  Should you wish to use a different token or if you are not using the Kubernetes controller, then you can 
override the Helm values `acls.env.CONSUL_HTTP_TOKEN` to set the name of the Kubernetes secret where your custom ACL token is stored.

```yaml title="Consul Release Controller Helm Values"
acls:
  enabled: false
  env:
  - name: CONSUL_HTTP_TOKEN
    valueFrom: 
      secretKeyRef:
        name: consul-controller-acl-token
        key: token 
```
#### TLS with auto encrypt

If Consul has been installed with the official Helm chart and you TLS enabled via auto encrypt using the following values:

```yaml title="Consul Release Controller Helm Values"
  tls:
    enabled: true
    enableAutoEncrypt: true
    httpsOnly: false
```

You can automatically configure the controller using the following config:

```yaml title="Consul Release Controller Helm Values"
autoEncrypt:
  enabled: false
```

If you are not using the default settings you can use the following Helm values to configure the chart to work correctly with your
installation.

```yaml
controller:
  enabled: "true"

  container_config:
    # Configure additional environment variables to be added to the controller container 
    env: []

    # Add additional volume mounts to the controller container. 
    additional_volume_mounts: []

    resources: {}

  # Add additional volumes to the controller deployment.
  additional_volumes: []

  # Add additional init containers to the controller deployment.
  additional_init_containers: []
```


</TabItem>
</Tabs>

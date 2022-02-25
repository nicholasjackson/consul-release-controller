---
sidebar_position: 2
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

# Docker based development environment

To create a local Docker based development environment that has all the prerequisites installed you can use [Shipyard](https://shipyard.run).  Shipyard is a simple tool that automates the creation of Docker containers, it can managed complex dependencies
such as ensuring a Kubernetes cluster exists before installing a Helm chart.

To run the example application, either first install shipyard ([https://shipyard.run/docs/install](https://shipyard.run/docs/install))
and then run the command below. Or you can use the bash script to install and create the environment in a single step.


<Tabs groupId="helm_values">
<TabItem value="noshipyard" label="Install and run">

```shell
curl https://shipyard.run/blueprint | \
  bash -s github.com/nicholasjackson/consul-release-controller//shipyard/docs_env
```

</TabItem>

<TabItem value="withshipyard" label="Shipyard already installed">
</TabItem>

</Tabs>

```shell
Running configuration from:  ./shipyard/docs_env

2022-02-25T07:50:09.167Z [INFO]  Creating resources from configuration: path=/home/nicj/go/src/github.com/nicholasjackson/consul-release-controller/shipyard/docs_env
2022-02-25T07:50:12.560Z [INFO]  Creating Output: ref=GRAFANA_PASSWORD
2022-02-25T07:50:12.560Z [INFO]  Creating Output: ref=TLS_CERT
2022-02-25T07:50:12.561Z [INFO]  Creating Output: ref=TEMPO_HTTP_ADDR
2022-02-25T07:50:12.561Z [INFO]  Creating Output: ref=KUBECONFIG

# ...

########################################################

Title Development setup
Author Nic Jackson

• Consul: https://localhost:8501
• Grafana: https://localhost:8080
• Application: http://localhost:18080

This blueprint defines 12 output variables.

You can set output variables as environment variables for your current terminal session using the following command:

eval $(shipyard env)

To list output variables use the command:

shipyard output


```

Once complete, you can set the environment variables needed to interact with the Kubernetes cluster and Consul using the
following command.

```shell
eval $(shipyard env)
```

If you now get pods with kubectl you will see that Consul and all the other software needed to test Consul Release Controller
has been installed for you.

```shell
➜ kubectl get pods --all-namespaces
NAMESPACE      NAME                                                   READY   STATUS    RESTARTS   AGE
kube-system    metrics-server-9cf544f65-x8ztm                         1/1     Running   0          4m31s
kube-system    local-path-provisioner-64ffb68fd-dnbk9                 1/1     Running   0          4m31s
kube-system    coredns-85cb69466-htxcl                                1/1     Running   0          4m31s
shipyard       connector-deployment-66c48db648-vl9gc                  1/1     Running   0          4m17s
cert-manager   cert-manager-cainjector-7974c84449-fpghx               1/1     Running   0          4m13s
cert-manager   cert-manager-77fd97f598-wbgkt                          1/1     Running   0          4m13s
cert-manager   cert-manager-webhook-59d6cfd784-ggxzc                  1/1     Running   0          4m13s
consul         consul-webhook-cert-manager-7cf6df6c4f-qn2p9           1/1     Running   0          4m4s
consul         consul-server-0                                        1/1     Running   0          4m4s
consul         consul-client-rq87z                                    1/1     Running   0          4m4s
consul         consul-controller-7569f96b56-2ht7n                     1/1     Running   0          4m4s
consul         consul-connect-injector-6f7cf9b878-sjrdn               1/1     Running   0          4m4s
monitoring     prometheus-kube-prometheus-operator-7498677577-9ghsb   1/1     Running   0          3m22s
monitoring     prometheus-prometheus-node-exporter-wnr48              1/1     Running   0          3m22s
monitoring     prometheus-kube-state-metrics-57c988498f-8rcvz         1/1     Running   0          3m22s
consul         consul-ingress-gateway-98786bbc6-qfksw                 2/2     Running   0          4m4s
monitoring     tempo-0                                                2/2     Running   0          3m7s
monitoring     prometheus-prometheus-kube-prometheus-prometheus-0     2/2     Running   0          3m18s
consul         consul-release-controller-788f459587-hnhbg             1/1     Running   0          3m4s
default        load-generator-deployment-85d566fd9d-8l7xf             2/2     Running   0          3m5s
monitoring     promtail-tqbbm                                         1/1     Running   0          3m9s
monitoring     grafana-79f678c684-qtdh6                               2/2     Running   0          3m5s
default        api-deployment-54dc89bcc9-tgg6j                        2/2     Running   0          3m5s
default        api-deployment-54dc89bcc9-xlnc9                        2/2     Running   0          3m5s
default        api-deployment-54dc89bcc9-t9k7m                        2/2     Running   0          3m5s
default        web-deployment-575887cd4c-gv7ng                        2/2     Running   0          3m5s
monitoring     loki-0                                                 1/1     Running   0          3m10s
```

## Cleaning up

To delete the environment you can use the following command:

```shell
shipyard destroy
```

This will remove all resources created by Shipyard.
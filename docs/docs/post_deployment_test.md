---
sidebar_position: 5
---

Post deployment tests allow the execution of HTTP requests against the candidate version of a service before it any production
traffic is sent to it.

When a post deployment test is defined, Consul Release Controller will expose an upstream target to the candidate, this allows
all tests to be executed over the service mesh with no manual routing changes required.

If the tests pass, the release controller will progress with the strategy, however, should the tests fail, the release controller
will roll back the deployment.

```yaml
---
apiVersion: consul-release-controller.nicholasjackson.io/v1
kind: Release
metadata:
  name: api
  namespace: default
spec:
  releaser:
    pluginName: "consul"
    config:
      consulService: "api"
  runtime:
    pluginName: "kubernetes"
    config:
      deployment: "api-deployment"
  postDeploymentTest:
    pluginName: "http"
    config:
      path: "/"
      method: "GET"
      requiredTestPasses: 3
      interval: "10s"
      timeout: "120s"
  strategy:
    pluginName: "canary"
    config:
      initialDelay: "30s"
      initialTraffic: 10
      interval: "30s"
      trafficStep: 20
      maxTraffic: 100
      errorThreshold: 5
  monitor:
    pluginName: "prometheus"
    config:
      address: "http://localhost:9090"
      queries:
        - name: "request-success"
          preset: "envoy-request-success"
          min: 99
        - name: "request-duration"
          preset: "envoy-request-duration"
          min: 20
          max: 200
```
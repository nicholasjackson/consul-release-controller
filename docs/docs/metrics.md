---
sidebar_position: 7
---

# Metrics

Before Consul Release Controller increases the traffic to your Candidate deployment it first checks the health of your application
by looking at the traffic metrics for the application. By default Consul Release Controller provides default queries for each of the 
supported platforms.

## Default Queries

### Prometheus / Kubernetes

#### EnvoyRequestSuccess

This query measures the HTTP response codes emitted from Envoy for your application and returns the percentage of requests (0-100)
that do not result in a HTTP 5xx response.

```javascript
sum(
	rate(
    envoy_cluster_upstream_rq{
      namespace="{{ .Namespace }}",
      pod=~"{{ .Name }}-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)",
      envoy_cluster_name="local_app",
      envoy_response_code!~"5.*"
    }[{{ .Interval }}]
  )
)
/
sum(
  rate(
    envoy_cluster_upstream_rq{
      namespace="{{ .Namespace }}",
      envoy_cluster_name="local_app",
      pod=~"{{ .Name }}-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)"
    }[{{ .Interval }}]
  )
)
* 100
```

#### EnvoyRequestDuration

This query measures the 99 percentile duration for application requests in milliseconds.

```javascript
histogram_quantile(
  0.99,
  sum(
    rate(
      envoy_cluster_upstream_rq_time_bucket{
        namespace="{{ .Namespace }}",
        envoy_cluster_name="local_app",
        pod=~"{{ .Name }}-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)"
      }[{{ .Interval }}]
    )
  ) by (le)
)
```

## Custom Queries

Custom queries can be defined by specifying the optional `query` parameter instead of the `preset` parameter.

Queries are specified as Prometheus queries and must return a numeric value that can be evaluated with the `min`, `max`
criteria. To enable generic queries Go templates can be used to inject values such as the `Name` of the deployment 
or the `Namespace` where the deployment is running.

```yaml
monitor:
  pluginName: "prometheus"
  config:
    address: "http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090"
    queries:
      - name: "mycustom"
        min: 20
        max: 200
        query: |
          histogram_quantile(
            0.99,
            sum(
              rate(
                envoy_cluster_upstream_rq_time_bucket{
                  namespace="{{ .Namespace }}",
                  envoy_cluster_name="local_app",
                  pod=~"{{ .Name }}-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)"
                }[{{ .Interval }}]
              )
            ) by (le)
          )
```

### Parameters

| Parameter       | Type        | Description                         |
| --------------- | ----------- | ----------------------------------- |
| Name            | string      | Name of the candidate deployment    |
| Namespace       | string      | Namespace where the candidate is running | 
| Interval        | duration    | Interval from the Strategy config, specified as a prometheus duration (30s, etc) |

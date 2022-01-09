package prometheus

const KubernetesEnvoyRequestSuccess = `
sum(
	rate(
    envoy_cluster_upstream_rq{
      namespace="{{ .Namespace }}",
      pod=~"{{ .Name }}-deployment-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)",
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
      pod=~"{{ .Name }}-deployment-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)"
    }[{{ .Interval }}]
  )
)
* 100
`

const KubernetesEnvoyRequestDuration = `
histogram_quantile(
  0.99,
  sum(
    rate(
      envoy_cluster_upstream_rq_time_bucket{
        namespace="{{ .Namespace }}",
        envoy_cluster_name="local_app",
        pod=~"{{ .Name }}-deployment-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)"
      }[{{ .Interval }}]
    )
  ) by (le)
)
`

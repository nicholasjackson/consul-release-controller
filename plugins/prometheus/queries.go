package prometheus

const KubernetesEnvoyRequestSuccess = `
sum(
	rate(
    envoy_cluster_upstream_rq{
      namespace="{{ .Namespace }}",
      pod!~"{{ .ReleaseName }}-primary.*",
      pod=~"{{ .CandidateName }}.*",
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
      pod!~"{{ .ReleaseName }}-primary.*",
      pod=~"{{ .CandidateName }}.*",
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
      	pod!~"{{ .ReleaseName }}-primary.*",
      	pod=~"{{ .CandidateName }}.*",
      }[{{ .Interval }}]
    )
  ) by (le)
)
`
const NomadEnvoyRequestSuccess = `
sum(
	rate(
    envoy_cluster_upstream_rq{
      job!~"{{ .ReleaseName }}-primary",
      job=~"{{ .CandidateName }}",
      envoy_cluster_name="local_app",
      envoy_response_code!~"5.*"
    }[{{ .Interval }}]
  )
)
/
sum(
  rate(
    envoy_cluster_upstream_rq{
      envoy_cluster_name="local_app",
      job!~"{{ .ReleaseName }}-primary",
      job=~"{{ .CandidateName }}",
    }[{{ .Interval }}]
  )
)
* 100
`

const NomadEnvoyRequestDuration = `
histogram_quantile(
  0.99,
  sum(
    rate(
      envoy_cluster_upstream_rq_time_bucket{
        envoy_cluster_name="local_app",
      	job!~"{{ .ReleaseName }}-primary",
      	job=~"{{ .CandidateName }}",
      }[{{ .Interval }}]
    )
  ) by (le)
)
`

prometheus_group_services = [{
  service_port_label = "http"
  service_name       = "prometheus"
  service_tags       = []
  sidecar_enabled    = true
  sidecar_upstreams  = []
  check_enabled      = true
  check_path         = "/-/healthy"
  check_interval     = "3s"
  check_timeout      = "1s"
}]

prometheus_task_app_prometheus_yaml = <<EOF
---
global:
  scrape_interval: 30s
  evaluation_interval: 3s
rule_files:
  - rules.yml
scrape_configs:
  - job_name: prometheus
    static_configs:
    - targets:
      - 0.0.0.0:9090
  - job_name: "nomad_server"
    metrics_path: "/v1/metrics"
    params:
      format:
      - "prometheus"
    consul_sd_configs:
    - server: "{{ env "attr.unique.network.ip-address" }}:8500"
      services:
        - "nomad"
      tags:
        - "http"
  - job_name: "nomad_client"
    metrics_path: "/v1/metrics"
    params:
      format:
      - "prometheus"
    consul_sd_configs:
    - server: "{{ env "attr.unique.network.ip-address" }}:8500"
      services:
        - "nomad-client"
  - job_name: "applications"
    metrics_path: "/metrics"
    params:
      format:
      - "prometheus"
    consul_sd_configs:
    - server: "{{ env "attr.unique.network.ip-address" }}:8500"
      tags:
        - "metrics"
    relabel_configs:
    - source_labels: [__meta_consul_service_metadata_job]
      target_label: job
    - source_labels: [__meta_consul_service_metadata_datacenter]
      target_label: datacenter
    
EOF
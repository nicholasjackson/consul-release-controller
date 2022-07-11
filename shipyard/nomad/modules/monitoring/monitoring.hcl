template "grafana_config" {
  source = <<-EOF
    upstreams = [{
      name = "prometheus"
      port = 9090
      },
      {
        name = "loki"
        port = 3100
    }]

    dashboards = [{
      name = "payments"
      data = <<EOT
      ${file("../example_app/jobs/payments-dashboard.json")}
      EOT
    }]

    custom_config = <<-EOT
      [auth.anonymous]
      enabled = true

      # Organization name that should be used for unauthenticated users
      org_name = Main Org.

      # Role for unauthenticated users, other valid values are `Editor` and `Admin`
      org_role = Viewer

      # Hide the Grafana version text from the footer and help tooltip for unauthenticated users (default: false)
      hide_version = true
    EOT

  EOF

  destination = "${data("monitoring_data")}/grafana.hcl"
}

template "prometheus_config" {
  source = <<-EOF
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
    
    prometheus_task_app_prometheus_yaml = <<EOT
    ---
    global:
      scrape_interval: 10s
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
    EOT
  EOF

  destination = "${data("monitoring_data")}/prometheus.hcl"
}

template "monitoring_pack" {
  source = <<-EOF
    #!/bin/bash -e

    nomad-pack run -f /scripts/prometheus.hcl /pack/prometheus
    nomad-pack run -f /scripts/grafana.hcl /pack/grafana
  EOF

  destination = "${data("monitoring_data")}/install_monitoring.sh"
}

exec_remote "pack" {
  depends_on = ["nomad_cluster.local", "template.monitoring_pack", "template.grafana_config"]

  image {
    name = "shipyardrun/hashicorp-tools:v0.9.0"
  }

  network {
    name = "network.dc1"
  }

  cmd = "/bin/bash"
  args = [
    "/scripts/install_monitoring.sh"
  ]

  # Mount a volume containing the config
  volume {
    source      = var.pack_folder
    destination = "/pack"
  }

  volume {
    source      = data("monitoring_data")
    destination = "/scripts"
  }

  working_directory = "/pack"

  env {
    key   = "NOMAD_ADDR"
    value = "http://server.local.nomad-cluster.shipyard.run:4646"
  }
}

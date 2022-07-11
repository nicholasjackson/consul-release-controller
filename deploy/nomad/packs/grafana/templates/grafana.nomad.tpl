job [[ template "job_name" . ]] {
  [[ template "region" . ]]
  datacenters = [[ .grafana.datacenters | toPrettyJson ]]

  // must have linux for network mode
  constraint {
    attribute = "${attr.kernel.name}"
    value     = "linux"
  }

  group "grafana" {
    count = 1

    network {
      mode = "bridge"

      port "http" {
        to = [[ .grafana.http_port ]]
      }
    }

    service {
      name = "grafana"
      port = "[[ .grafana.http_port ]]"

      connect {
        sidecar_service {
          proxy {
            [[ range $upstream := .grafana.upstreams ]]
            upstreams {
              destination_name = [[ $upstream.name | quote ]]
              local_bind_port  = [[ $upstream.port ]]
            }
            [[ end ]]
          }
        }
      }
    }

    task "grafana" {
      driver = "docker"

      config {
        image = "grafana/grafana:[[ .grafana.version_tag ]]"
        volumes = [
          "local/datasources:/etc/grafana/provisioning/datasources",
          "local/provisioning_dashboards:/etc/grafana/provisioning/dashboards",
          "local/dashboards:/etc/dashboards",
          "local/config/custom.ini:/etc/grafana/grafana.ini",
        ]
      }

      resources {
        cpu    = [[ .grafana.resources.cpu ]]
        memory = [[ .grafana.resources.memory ]]
      }

      template {
        change_mode   = "signal"
        change_signal = "SIGHUP"
        destination   = "local/provisioning_dashboards/main.yml"
        data = <<-EOF
          apiVersion: 1

          providers:
            - name: dashboards
              type: file
              updateIntervalSeconds: 30
              options:
                path: /etc/dashboards
        EOF
      }
      
      template {
        change_mode   = "signal"
        change_signal = "SIGHUP"
        destination   = "local/config/custom.ini"
        data = <<-EOF
[[ .grafana.custom_config ]]
        EOF
      }

      [[- if .grafana.datasources ]]
      [[- range $idx, $datasource := .grafana.datasources ]]
      template {
        change_mode   = "signal"
        change_signal = "SIGHUP"
        destination   = "local/datasources/[[ $datasource.name ]].yml"
        data = <<EOF
[[ $datasource.data ]]
        EOF
      }
      [[- end]]
      [[- end]]

      [[- if .grafana.dashboards ]]
      [[- range $idx, $dashboard := .grafana.dashboards ]]
      template {
        change_mode   = "signal"
        change_signal = "SIGHUP"
        left_delimiter = "#{{"
        right_delimiter = "}}"
        destination   = "local/dashboards/[[ $dashboard.name ]].json"
        data = <<EOF
[[ $dashboard.data ]]
        EOF
      }
      [[- end]]
      [[- end]]
    }
  }
}

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
          "local/dashboards:/etc/grafana/provisioning/dashboards",
        ]
      }

      resources {
        cpu    = [[ .grafana.resources.cpu ]]
        memory = [[ .grafana.resources.memory ]]
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
    }
  }
}

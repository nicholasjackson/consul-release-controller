job "waypoint-runner" {
  type = "service"

  datacenters = ["dc1"]

  group "waypoint-runner" {
    count = 1

    network {
      mode = "bridge"
    }

    volume "data" {
      type   = "host"
      source = "waypoint"
    }

    task "runner" {
      driver = "docker"

      template {
        data = <<-EOH
          #!/bin/sh

          until [ -f /data/waypoint.token ]
          do
            sleep 5
          done

          export NOMAD_ADDR="http://{{env "attr.unique.network.ip-address"}}:4646"
          export WAYPOINT_SERVER_ADDR="{{env "attr.unique.network.ip-address"}}:9701"
          export WAYPOINT_SERVER_TLS="true"
          export WAYPOINT_SERVER_TLS_SKIP_VERIFY="true"
          export WAYPOINT_SERVER_TOKEN="$(cat /data/waypoint.token)"

          waypoint runner agent -vv
        EOH

        destination = "local/runner.sh"
      }

      config {
        image = "hashicorp/waypoint:latest"

        entrypoint = [""]
        command    = "sh"
        args       = ["/scripts/runner.sh"]

        volumes = [
          # Use named volume created outside nomad.
          "local:/scripts"
        ]
      }

      volume_mount {
        volume      = "data"
        destination = "/data"
      }

      resources {
        cpu    = 500 # MHz
        memory = 128 # MB
      }
    }
  }
}
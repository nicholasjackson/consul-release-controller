job "waypoint-server" {
  type = "service"

  datacenters = ["dc1"]

  group "waypoint-server" {
    count = 1

    volume "data" {
      type   = "host"
      source = "waypoint"
    }

    network {
      mode = "bridge"

      port "ui" {
        static = "9702"
        to     = "9702"
      }

      port "server" {
        static = "9701"
        to     = "9701"
      }
    }

    service {
      name = "waypoint-ui"
      port = "ui"
      tags = ["waypoint"]
    }

    service {
      name = "waypoint-server"
      port = "server"
      tags = ["waypoint"]
    }

    task "server" {
      driver = "docker"

      config {
        ports = ["ui", "server"]
        image = "hashicorp/waypoint:latest"

        args = [
          "server",
          "run",
          "-accept-tos",
          "-vv",
          "-db=/home/waypoint/data.db",
          "-listen-grpc=0.0.0.0:9701",
          "-listen-http=0.0.0.0:9702"
        ]
      }

      resources {
        cpu    = 500 # MHz
        memory = 128 # MB
      }
    }

    task "bootstrap" {
      driver = "docker"

      lifecycle {
        hook = "poststart"
      }

      template {
        data = <<-EOH
          #!/bin/sh

          # Only bootstrap if the token does not exist 
          if [ -f /data/waypoint.token ]; then
            exit 0
          fi

          waypoint \
          server \
          bootstrap \
          -server-addr=127.0.0.1:9701 \
          -server-tls-skip-verify \
          > /data/waypoint.token

          cat <<-EOF > /data/runner.hcl
          nomad_host="http://{{env "attr.unique.network.ip-address"}}:4646"

          EOF

          waypoint runner profile set \
            -name=nomad \
            -plugin-type=nomad \
            -env-var="NOMAD_ADDR=http://{{env "attr.unique.network.ip-address"}}:4646" \
            -env-var="WAYPOINT_SERVER_ADDR={{env "attr.unique.network.ip-address"}}:9701" \
            -env-var="WAYPOINT_SERVER_TLS=true" \
            -env-var="WAYPOINT_SERVER_TLS_SKIP_VERIFY=true" \
            -oci-url="[[ .waypoint.waypoint_odr_image ]]" \
            -plugin-config=/data/runner.hcl \
            -default

          echo "# Waypoint Token"
          cat /data/waypoint.token

          echo "# To create a local context"

          echo "
          waypoint context create \\
            -server-addr=localhost:9701 \\
            -server-auth-token=$(cat /data/waypoint.token) \\
            -server-require-auth=true -server-tls-skip-verify \\
            -set-default localhost-ui"

          echo "
          waypoint context create \\
            -server-addr=localhost:9701 \\
            -server-auth-token=$(cat /data/waypoint.token) \\
            -server-require-auth=true -server-tls-skip-verify \\
            -set-default localhost-ui" > /data/create_context.sh

          chmod +x /data/create_context.sh
        EOH

        destination = "local/bootstrap.sh"
      }

      config {
        image = "hashicorp/waypoint:latest"

        entrypoint = [""]
        command    = "sh"
        args       = ["/scripts/bootstrap.sh"]

        volumes = [
          # Use named volume created outside nomad.
          "local:/scripts"
        ]
      }

      volume_mount {
        volume      = "data"
        destination = "/data"
      }
    }
  }
}
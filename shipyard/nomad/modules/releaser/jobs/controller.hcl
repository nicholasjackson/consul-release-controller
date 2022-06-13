job "release-controller" {
  type = "service"

  datacenters = ["dc1"]

  group "release-controller" {
    count = 1

    network {
      mode = "bridge"
      port "http" {
        static = "8080"
        to     = "8080"
      }
    }

    service {
      name = "consul-release-controller"
      port = "8080"

      connect {
        sidecar_service {
          proxy {
            upstreams {
              destination_name = "consul-release-controller-upstreams"
              local_bind_port  = 18080
            }
          }
        }
      }
    }

    task "socat" {
      driver = "docker"

      config {
        ports = ["http"]
        image = "alpine/socat"
        args = [
          "TCP-LISTEN:8080,fork",
          "TCP:127.0.0.1:18080",
        ]
      }

      resources {
        cpu    = 500 # MHz
        memory = 128 # MB
      }
    }
  }
}
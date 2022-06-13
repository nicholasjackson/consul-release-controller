job "api-deployment" {
  type = "service"

  datacenters = ["dc1"]

  group "api" {
    count = 3

    network {
      mode = "bridge"
      port "http" {
        to = "9090"
      }

      # dynamic port for the metrics
      port "metrics" {
        to = "9102"
      }
    }

    # create a service so that promethues can scrape the metrics
    service {
      name = "api-metrics"
      port = "metrics"
      tags = ["metrics"]
      meta {
        metrics    = "prometheus"
        job        = "${NOMAD_JOB_NAME}"
        datacenter = "${node.datacenter}"
      }
    }

    service {
      name = "api"
      port = "9090"

      connect {
        sidecar_service {
          proxy {
            # expose the metrics endpont 
            config {
              envoy_prometheus_bind_addr = "0.0.0.0:9102"
            }

            upstreams {
              destination_name = "payments"
              local_bind_port  = 9091
            }
          }
        }
      }
    }


    task "api" {
      driver = "docker"

      config {
        image = "nicholasjackson/fake-service:v0.23.1"
        ports = ["http"]
      }

      env {
        NAME          = "API V1"
        UPSTREAM_URIS = "http://localhost:9091"
      }

      resources {
        cpu    = 500 # MHz
        memory = 128 # MB
      }
    }

    # envoy is bound to ip 127.0.0.2 however expose only accepts redirects to 127.0.0.1
    # run socat to redirect the envoy admin port to localhost
    task "socat" {
      driver = "docker"

      config {
        image = "alpine/socat"
        args = [
          "TCP-LISTEN:19002,fork,bind=127.0.0.1",
          "TCP:127.0.0.2:19002",
        ]
      }
    }
  }
}
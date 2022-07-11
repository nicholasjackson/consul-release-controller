job "ingress" {

  type = "system"

  datacenters = ["dc1"]

  group "ingress" {

    network {
      mode = "bridge"
      port "http" {
        static = 18080
        to     = 18080
      }

      port "metrics" {
        to = "9102"
      }
    }

    service {
      name = "ingress-metrics"
      port = "metrics"
      tags = ["metrics"]
      meta {
        metrics    = "prometheus"
        job        = NOMAD_JOB_NAME
        datacenter = node.datacenter
      }
    }

    service {
      name = "ingress"
      port = "18080"

      connect {
        gateway {
          proxy {
            # expose the metrics endpont 
            config {
              envoy_prometheus_bind_addr = "0.0.0.0:9102"
            }
          }

          ingress {
            listener {
              port     = 18080
              protocol = "http"

              service {
                name  = "grafana"
                hosts = ["grafana.ingress.shipyard.run", "grafana.hashiconf.hashicraft.com"]
              }

              service {
                name  = "prometheus"
                hosts = ["prometheus.ingress.shipyard.run", "prometheus.hashiconf.hashicraft.com"]
              }

              service {
                name  = "consul-release-controller"
                hosts = ["releaser.ingress.shipyard.run", "releaser.hashiconf.hashicraft.com"]
              }

              service {
                name  = "api"
                hosts = ["*"]
              }
            }

          }
        }
      }
    }
  }
}

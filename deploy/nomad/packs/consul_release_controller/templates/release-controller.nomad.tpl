job "release-controller" {
  type = "service"

  datacenters = ["dc1"]

  group "release-controller" {
    count = 1

    network {
      mode = "bridge"
      
      port "http" {
        to = 8080
      }
    }

    service {
      name = "consul-release-controller"
      port = "8080"

      connect {
        sidecar_service {
          proxy {
            upstreams {
              destination_name = "prometheus"
              local_bind_port = 9090
            }
          } 
        }
      }
    }

    task "release-controller" {
      driver = "docker"

      template {
        data = <<-EOH
[[ .consul_release_controller.tls_cert ]]
        EOH

        destination = "local/certs/cert.pem"
      }
      
      template {
        data = <<-EOH
[[ .consul_release_controller.tls_key ]]
        EOH

        destination = "local/certs/key.pem"
      }

      config {
        image = [[ .consul_release_controller.controller_image | quote]]
        
        volumes = [
          # Use named volume created outside nomad.
          "local/certs:/certs"
        ]
      }

      env {
        ENABLE_NOMAD = "true"  
        TLS_CERT = "/certs/cert.pem" 
        TLS_KEY = "/certs/key.pem" 
        CONSUL_HTTP_ADDR = [[ .consul_release_controller.consul_http_addr | quote]]
        NOMAD_ADDR = [[ .consul_release_controller.nomad_addr | quote]]
        HTTP_API_BIND_ADDRESS = "0.0.0.0"
      }

      resources {
        cpu    = 500 # MHz
        memory = 128 # MB
      }
    }
  }
}

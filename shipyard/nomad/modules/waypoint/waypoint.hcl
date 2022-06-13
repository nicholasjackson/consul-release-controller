# script to download custom waypoint plugins and add them to the odr image
template "download_plugin" {
  source = <<-EOF
    # Determine the OS and architecure
    ARCH=$(uname -m)
    SHIPYARD_ARCH="amd64"

    if [ "$${ARCH}" == "x86_64" ]; then
      SHIPYARD_ARCH="amd64"
    fi

    if [ "$${ARCH}" == "arm64" ]; then
      SHIPYARD_ARCH="arm64"
    fi

    wget --no-check-certificate https://github.com/hashicorp-dev-advocates/waypoint-plugin-noop/releases/download/v0.2.2/waypoint-plugin-noop_linux_$${SHIPYARD_ARCH}.zip && \
      unzip waypoint-plugin-noop_linux_$${SHIPYARD_ARCH}.zip
  EOF

  destination = "${data("waypoint_data")}/download_plugin.sh"
}

# build a custom waypoint ODR image that contains the insecure registry certs
template "waypoint_odr" {
  source = <<-EOF
    FROM hashicorp/waypoint-odr:latest
    SHELL ["/kaniko/bin/sh", "-c"]

    ENV HOME /root
    ENV USER root
    ENV PATH="$${PATH}:/kaniko"
    ENV SSL_CERT_DIR=/kaniko/ssl/certs
    ENV DOCKER_CONFIG /kaniko/.docker/
    ENV XDG_CONFIG_HOME=/kaniko/.config/
    ENV TMPDIR /kaniko/tmp
    ENV container docker

    COPY ./root.cert  /kaniko/ssl/certs/registry.pem
    RUN cat /kaniko/ssl/certs/registry.pem >> /kaniko/ssl/certs/ca-certificates.crt

    COPY ./download_plugin.sh /kaniko/bin/download_plugin.sh
    RUN ls -las /kaniko/bin
    RUN sh /kaniko/bin/download_plugin.sh

    RUN mkdir -p /kaniko/.config/waypoint/plugins/
    RUN cp waypoint-plugin-noop /kaniko/.config/waypoint/plugins/waypoint-plugin-noop
  EOF

  destination = "${data("waypoint_data")}/Dockerfile.odr"
}

# copy the certs for the registry 
template "copy_certs" {
  source = <<-EOF
    #!/bin/sh

    cp ${file_dir()}/certs/* ${data("waypoint_data")}
  EOF

  destination = "${data("waypoint_data")}/copy_certs.sh"
}

exec_local "copy_certs" {
  depends_on = ["template.copy_certs"]
  cmd        = "sh"
  args = [
    "copy_certs.sh",
  ]

  working_directory = data("waypoint_data")
}

# If this tag is updated then the waypoint-server job needs the corresponding change
variable "waypoint_odr_tag" {
  default = "0.0.7"
}

# If this tag is updated then the waypoint-server job needs the corresponding change
variable "cn_nomad_load_image" {
  default = "shipyard.run/localcache/waypoint-odr:0.0.7"
}

# Build a custom ODR with our certs
container "waypoint-odr" {
  depends_on = ["exec_local.copy_certs"]

  network {
    name = "network.dc1"
  }

  build {
    file    = "./Dockerfile.odr"
    context = data("waypoint_data")
    tag     = var.waypoint_odr_tag
  }

  command = ["/kaniko/waypoint"]
}


container "registry" {
  depends_on = ["exec_local.copy_certs"]
  network {
    name       = "network.dc1"
    ip_address = "10.5.0.100"
  }

  image {
    name = "registry:2"
  }

  volume {
    source      = data("waypoint_data")
    destination = "/certs"
  }

  env_var = {
    REGISTRY_HTTP_ADDR            = "0.0.0.0:443"
    REGISTRY_HTTP_TLS_CERTIFICATE = "/certs/leaf.cert"
    REGISTRY_HTTP_TLS_KEY         = "/certs/leaf.key"
  }

  port {
    local  = 443
    remote = 443
    host   = 443
  }
}

# Override the Docker config to add the custom registry
variable "cn_nomad_docker_insecure_registries" {
  default = ["10.5.0.100"]
}

template "waypoint-pack" {
  source = <<-EOF
    #!/bin/sh
    
    nomad-pack run \
      --var="waypoint_odr_image=${var.cn_nomad_load_image}" \
      /pack/nomad-pack-community-registry-main/packs/waypoint
  EOF

  destination = "${data("waypoint_data")}/install_waypoint.sh"
}

exec_remote "waypoint_pack" {
  depends_on = ["nomad_cluster.local"]

  image {
    name = "shipyardrun/hashicorp-tools:v0.8.0"
  }

  network {
    name = "network.dc1"
  }

  cmd = "/bin/bash"
  args = [
    "/scripts/install_waypoint.sh"
  ]

  # Mount a volume containing the config
  volume {
    source      = "${file_dir()}/../../pack"
    destination = "/pack"
  }

  volume {
    source      = data("waypoint_data")
    destination = "/scripts"
  }

  working_directory = "/pack"

  env {
    key   = "NOMAD_ADDR"
    value = "http://server.local.nomad-cluster.shipyard.run:4646"
  }
}

output "WAYPOINT_TOKEN" {
  value = "$(cat ${data("waypoint")}/waypoint.token)"
}

nomad_ingress "waypoint-ui" {
  cluster = var.cn_nomad_cluster_name
  job     = "waypoint-server"
  group   = "waypoint-server"
  task    = "server"

  network {
    name = "network.dc1"
  }

  port {
    local  = 9702
    remote = "ui"
    host   = 9702
  }
}

nomad_ingress "waypoint-server" {
  cluster = var.cn_nomad_cluster_name
  job     = "waypoint-server"
  group   = "waypoint-server"
  task    = "server"

  network {
    name = "network.dc1"
  }

  port {
    local  = 9701
    remote = "server"
    host   = 9701
  }
}
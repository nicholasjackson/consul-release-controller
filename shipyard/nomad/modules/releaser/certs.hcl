certificate_ca "releaser_ca" {
  output = data("nomad_config")
}

certificate_leaf "releaser_leaf" {
  depends_on = ["certificate_ca.releaser_ca"]

  ca_cert = "${data("nomad_config")}/releaser_ca.cert"
  ca_key  = "${data("nomad_config")}/releaser_ca.key"

  ip_addresses = ["127.0.0.1"]
  dns_names    = ["127.0.0.1:9443"]

  output = data("nomad_config")
}

output "TLS_KEY" {
  value = "${data("nomad_config")}/releaser_leaf.key"
}

output "TLS_CERT" {
  value = "${data("nomad_config")}/releaser_leaf.cert"
}

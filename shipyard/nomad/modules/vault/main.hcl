variable "vault_bootstrap_script" {
  default = <<-EOF
  #/bin/sh -e
  vault status
  EOF
}

variable "vault_network" {
  default = var.cn_network
}

module "vault" {
  source = "github.com/shipyard-run/blueprints?ref=f235847a73c5bb81943aaed8f0c526edee693d75/modules//vault-dev"
}

output "VAULT_ADDR" {
  value = "http://localhost:8200"
}

output "VAULT_TOKEN" {
  value = "root"
}
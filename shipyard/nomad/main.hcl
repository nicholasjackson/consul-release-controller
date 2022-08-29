network "dc1" {
  subnet = "10.5.0.0/16"
}

module "consul_nomad" {
  disabled = !var.install_nomad

  //depends_on = ["container.waypoint-odr"]
  source = "github.com/shipyard-run/blueprints?ref=694e825167a05d6ae035a0b91f90ee7e8b2d2384/modules//consul-nomad"
}

module "monitoring" {
  disabled = !var.install_monitoring

  depends_on = ["module.consul_nomad"]
  source     = "./modules/monitoring"
}

module "example_app" {
  disabled = !var.install_example_app

  source = "./modules/example_app"
}

module "controller" {
  disabled = var.install_controller == ""

  source = "./modules/releaser"
}

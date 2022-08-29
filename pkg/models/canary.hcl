deployment "api" {
  // name of the consul service to which the deployment relates
  consul_service = "api"

  kubernetes_workload {
    // name of the deployment to monitor
    deployment = "api"
  }

  strategy_canary {
    // how often the analysis is run
    // default: 1m
    interval = "30s"

    // initial percentage of traffic to route to the canary
    // default: 5
    initial_traffic = 10

    // percentage of traffic to increase with every step
    // default: 5
    traffic_step = 10

    // maximum percentage before promoting to primary 
    // default: 100
    max_traffic = 100

    // number of failed checks before rolling back the canary
    // default: 5
    error_threshold = 5

    // should the canary workload be removed on a failed check  
    // default: true
    delete_canary_on_failed = false

    // should the canary be automatically promoted
    automatic_promotion = true

    check {
      metric = "request-success"
      // 99% of requests must be successful
      min = 99
    }

    check {
      metric = "request-duration"
      // 20ms is the fastest response time
      min = 20
      // 200ms is the slowest response time
      max = 200
    }
  }
}

metric "request-success" {
  type = "prometheus"

  query = <<EOF

  EOF
}

metric "request-duration" {
  type = "prometheus"

  query = <<EOF

  EOF
}
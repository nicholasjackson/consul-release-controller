---
title: Consul Release Controller - Nomad
author: Nic Jackson
slug: release_controller_nomad
---

## Consul UI

```
http://consul.container.shipyard.run:18500/ui/dc1/services
```

## Nomad

To use the nomad CLI set the Shipyard environment variables using the following command:

```
eval $(shipyard env)
```

This will set the `NOMAD_ADDR`, since Nomad is running on a dynamic port, if you would like to open the UI you can
use the location defined by this environment variable.

```
echo $NOMAD_ADDR
xdg-open $NOMAD_ADDR
```

# Creating a new release

An example application has been installed on Nomad consting of a two tier application `api -> payments`. Both 
applictaions are running on Consul service mesh and the `api` is exposed using Consul Ingress Gateway. To simulate production
load, a simple load generation script is applying 10 requests per second to the application ingress.

The files for the application are located at `./modules/example_app/jobs`

You can test the example app by curling it from your terminal.

```shell
➜ curl api.ingress.shipyard.run
{
  "name": "API V1",
  "uri": "/",
  "type": "HTTP",
  "ip_addresses": [
    "172.26.64.4"
  ],
  "start_time": "2022-07-11T13:03:29.852019",
  "end_time": "2022-07-11T13:03:29.874086",
  "duration": "22.066884ms",
  "body": "Hello World",
  "upstream_calls": {
    "http://localhost:3001": {
      "name": "PAYMENTS V1",
      "uri": "http://localhost:3001",
      "type": "HTTP",
      "ip_addresses": [
        "172.26.64.10"
      ],
      "start_time": "2022-07-11T13:03:29.853007",
      "end_time": "2022-07-11T13:03:29.873269",
      "duration": "20.261488ms",
      "headers": {
        "Content-Length": "263",
        "Content-Type": "text/plain; charset=utf-8",
        "Date": "Mon, 11 Jul 2022 13:03:29 GMT",
        "Server": "envoy",
        "X-Envoy-Upstream-Service-Time": "21"
      },
      "body": "Hello World",
      "code": 200
    }
  },
  "code": 200
}
```

To create a new release you first need to configure it by posting the configuration to the release controller API.

```shell
curl releaser.ingress.shipyard.run/v1/releases -d @./modules/example_app/jobs/valid_nomad_releae.json
```

## Monitoring the relase status

To monitor the status of the release you can use the API to get the status of the current release

```shell
watch 'curl -s releaser.ingress.shipyard.run/v1/releases | jq .'
```

## Application Dashboard

If you open the Grafana dashboard you will see the traffic flowing to the primary and canary version of the example app. Once the release
has been configured you should see 100% traffic flowing to the Primary version.

```
http://grafana.ingress.shipyard.run/d/payments/application-dashboard?orgId=1&refresh=10s
```

## Creating a new release

To create a new release you can re-deploy the payments application.

```shell
nomad run ./modules/example_app/jobs/payments.hcl
```

Once deployed the release controller will start monitoring the status of the application, it will then start to slow introduce traffic to the new version.
If you continue to monitor the Grafana dashboard, you will see the `Payments Request per Second` move from `Primary` to `Canary`. Once 100% of traffic
has been applied to the `Canary`, the release controller will replace the current `Primary` with the `Canary` and re-apply 100% of traffic to the `Primary`. The systems is now ready to accept another deployment.

## Creating a release with rollback

To simulate a failed relase you can use the following job.

```shell
nomad run ./modules/example_app/jobs/payments_with_error.hcl
```

The release controller will detect a fault with this release and will roll back the deployment. To add fault tollerance to this job failure apply the consul configuration to add a retry to the payments requests.

```shell
consul config write ./modules/example_app/consul_config/payments_retry.hcl
```

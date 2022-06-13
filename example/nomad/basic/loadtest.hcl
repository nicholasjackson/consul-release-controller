job "loadtest" {
  type = "service"

  datacenters = ["dc1"]

  group "loadtest" {
    count = 1

    network {
      mode = "bridge"
    }

    service {
      name = "loadtest"
    }

    task "loadtest" {
      driver = "docker"

      config {
        image = "loadimpact/k6"
        args = [
          "run",
          "/etc/config/load_test.js",
        ]

        mount {
          type   = "bind"
          source = "local"
          target = "/etc/config"
        }
      }

      env {
        NAME          = "K6_STATSD_ADDR"
        UPSTREAM_URIS = "localhost:9125"
      }

      template {
        data = <<-EOH
          import http from 'k6/http';
          import { sleep, check } from 'k6';
          import { Counter } from 'k6/metrics';
          import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.1.0/index.js';

          // A simple counter for http requests
          export const requests = new Counter('http_reqs');
          // you can specify stages of your test (ramp up/down patterns) through the options object
          // target is the number of VUs you are aiming for
          export const options = {
            vus: 10,
            duration: '30m',
          };

            //maxVUs: 10,
            //startRate: 1,
            //timeUnit: '1s',
            //stages: [
            //  { target: 1, duration: '59s' },
            //  { target: 10, duration: '120s' },
            //  { target: 10, duration: '60s' },
            //  { target: 0, duration: '1s' },
            //  { target: 0, duration: '59s' },
            //],
          export default function () {
            var payload = 'Replicants are like any other machine, are either a benefit or a hazard'
            var params = {
              headers: {
                'Content-Type': 'text/plain',
              },
            }
            // our HTTP request, note that we are saving the response to res, which can be accessed later
            const res = http.get('http://{{ env "attr.unique.network.ip-address" }}:18080');
            const checkRes = check(res, {
              'status is 200': (r) => r.status === 200,
            });

            sleep(randomIntBetween(0.2, 1.5));
          }
        EOH

        destination = "local/load_test.js"
      }
    }
  }
}
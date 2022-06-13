#!/bin/sh -e

until curl -s ${CONSUL_HTTP_ADDR}/v1/status/leader | grep 8300; do
  echo "Waiting for Consul to start"
  sleep 1
done

consul config write ./proxy-defaults.hcl
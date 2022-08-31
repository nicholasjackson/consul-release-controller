---
replicaCount: 1

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

controller:
  enabled: "true"

  container_config:
    image:
      repository: nicholasjackson/consul-release-controller
      pullPolicy: IfNotPresent
      # Overrides the image tag whose default is the chart appVersion.
      tag: "##VERSION##"

    # Set the CONSUL_HTTP_ADDR to the address of the Consul cluster, this env is only used when
    # autoEncrypt is disabled as autoEncrypt adds a https variable.
    hostEnv:
    - name: HOST_IP
      valueFrom:
        fieldRef:
          fieldPath: status.hostIP
    - name: CONSUL_HTTP_ADDR
      value: http://$(HOST_IP):8500

    # Configure additional environment variables to be added to the controller container
    env: []

    # Add additional volume mounts to the controller container.
    additional_volume_mounts: []

    resources: {}

  # Add additional volumes to the controller deployment.
  additional_volumes: []

  # Add additional init containers to the controller deployment.
  additional_init_containers: []

  podAnnotations: {}

  podSecurityContext: {}

  securityContext: {}

  autoscaling:
    enabled: false
    minReplicas: 1
    maxReplicas: 100
    targetCPUUtilizationPercentage: 80
    # targetMemoryUtilizationPercentage: 80

  nodeSelector: {}

  tolerations: []

  affinity: {}

serviceAccount:
  # Specifies whether a service account should be created
  create: true
  # Annotations to add to the service account
  annotations: {}
  # The name of the service account to use.
  # If not set and create is true, a name is generated using the fullname template
  name: ""

# Default values when Consul has been deployed using the Helm chart with autoencrypt
autoEncrypt:
  enabled: false
  env:
    - name: CONSUL_CAPATH
      value: /consul/tls/client/ca/tls.crt
    - name: HOST_IP
      valueFrom:
        fieldRef:
          fieldPath: status.hostIP
    - name: CONSUL_HTTP_ADDR
      value: https://$(HOST_IP):8501

  controller_volume_mounts:
  - mountPath: /consul/tls/client/ca
    name: consul-auto-encrypt-ca-cert

  additional_volumes:
    - name: consul-server-ca
      secret:
        secretName: consul-server-cert
    - name: consul-auto-encrypt-ca-cert
      emptyDir:
        medium: Memory

  #  Add a custom init container to fetch the client certificate
  init_container:
    - command:
      - /bin/sh
      - -ec
      - |
        consul-k8s get-consul-client-ca \
          -output-file=/consul/tls/client/ca/tls.crt \
          -server-addr=consul-server \
          -server-port=8501 \
          -ca-file=/consul/tls/ca/tls.crt
      image: hashicorp/consul-k8s:0.25.0
      imagePullPolicy: IfNotPresent
      name: get-auto-encrypt-client-ca
      resources:
        limits:
          cpu: 50m
          memory: 50Mi
        requests:
          cpu: 50m
          memory: 50Mi
      volumeMounts:
      - mountPath: /consul/tls/ca
        name: consul-server-ca
      - mountPath: /consul/tls/client/ca
        name: consul-auto-encrypt-ca-cert

# If ACLs are enabled a secret containing a valid ACL token is required
acls:
  secretName: ""
  secretKey: ""

# Configures Prometheus metrics collection
prometheus:
  # Enable Prometheus metrics collection, Helm configures Prometheus ServiceMonitor to scape internal metrics
  enabled: true
  # Namespace where Prometheus is deployed
  namespace: monitoring
  # Interval for Prometheus to scrape the controller
  scrapeInterval: 15s

# Not currently used, conversion webhooks will be eventually enabled
webhook:
  enabled: "true"
  type: ClusterIP
  port: 443

  # Override the default webhook service name and namespace
  # This can be used when running the controller locally
  service: ""
  namespace: ""


  # Allows adding additional DNS Names to the cert generated
  # for the webhook
  additionalDNSNames: []

  failurePolicy: Fail

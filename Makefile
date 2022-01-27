DOCKER_REGISTRY ?= docker.io/nicholasjackson
HELM_VERSION ?= 0.0.1
SHELL := /bin/bash
UNAME := $(shell uname)

ifeq "$(VERSION)"  ""
	VERSION = $(shell git log --pretty=format:'%h' -n 1)
else 
	VERSION = $(VERSION)
endif

# Build and push the Arm64 and x64 images to the Docker registry
build_docker:
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	docker buildx create --name multi || true
	docker buildx use multi
	docker buildx inspect --bootstrap
	docker buildx build --platform linux/arm/v6,linux/arm/v7,linux/arm64,linux/amd64 \
		-t ${DOCKER_REGISTRY}/consul-release-controller:${VERSION} \
    -f ./Dockerfile \
    .  \
		--push
	docker buildx rm multi

# Build a x64 images and import into the local registry
build_docker_dev:
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	docker buildx create --name multi || true
	docker buildx use multi
	docker buildx inspect --bootstrap
	docker buildx build --platform linux/amd64 \
		-t ${DOCKER_REGISTRY}/consul-release-controller:${VERSION}.dev \
    -f ./Dockerfile \
    . \
		--load
	docker buildx rm multi

# Fetch Kubernetes certificates needed to secure the server with TLS
fetch_kubernetes_certs:
	mkdir -p /tmp/k8s-webhook-server/serving-certs/
	
	kubectl get secret consul-release-controller-webhook-certificate -n consul -o json | \
		jq -r '.data."tls.crt"' | \
		base64 -d > /tmp/k8s-webhook-server/serving-certs/tls.crt
	
	kubectl get secret consul-release-controller-webhook-certificate -n consul -o json | \
		jq -r '.data."tls.key"' | \
		base64 -d > /tmp/k8s-webhook-server/serving-certs/tls.key

run_kubernetes: fetch_kubernetes_certs
	go run main.go

# Create the shipyard environment and run the functional tests
functional_tests_kubernetes:
	cd functional_tests && go run .

# Run the functional tests, without creating the environment
# the environment can be created manually by running shipyard run ./shipyard/kubernetes
functional_tests_kubernetes_no_env:
	cd functional_tests && go run . --create-environment=false

# Create a new release for kubernetes
deploy_kubernetes_relase:
	curl -k https://localhost:9443/v1/releases -XPOST -d @./example/kubernetes/canary/api.json

create_dev_env_local_controller:
	shipyard run ./shipyard/kubernetes --var="controller_enabled=false"

create_dev_env_docker_controller:
	shipyard run ./shipyard/kubernetes --var="controller_enabled=true" --var="controller_version=${VERSION}.dev" 

generate_helm:
	cd ./kubernetes/controller && make manifests
	cd ./kubernetes/controller && make generate

# First generate the Helm specific kustomize config that creates the RBAC and CRDs
	kustomize build ./kubernetes/controller/config/helm -o ./deploy/kubernetes/charts/consul-release-controller/templates

# Set the version in the chart
	cp ./deploy/kubernetes/charts/consul-release-controller/Chart.tpl ./deploy/kubernetes/charts/consul-release-controller/Chart.yaml
	sedi=(-i) && [ "$(UNAME)" == "Darwin" ] && sedi=(-i '') ; \
	sed "$${sedi[@]}" -e 's/##VERSION##/${HELM_VERSION}/' ./deploy/kubernetes/charts/consul-release-controller/Chart.yaml

# Now package the Helm chart into a tarball
	helm package ./deploy/kubernetes/charts/consul-release-controller

# Move it to the ./docs folder used to serve Github Pages
	mv consul-release-controller-${HELM_VERSION}.tgz ./docs/

# Generate the index
	cd ./docs && helm repo index .
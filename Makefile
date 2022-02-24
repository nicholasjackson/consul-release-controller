DOCKER_REGISTRY ?= docker.io/nicholasjackson
SHELL := /bin/bash
UNAME := $(shell uname)

ifeq "$(VERSION_ENV)" ""
	VERSION=$(shell git log --pretty=format:'%h' -n 1)
	HELM_VERSION=0.0.3-dev
else 
	VERSION=$(VERSION_ENV)
	HELM_VERSION=$(VERSION_ENV)
endif

# Build and push the Arm64 and x64 images to the Docker registry
build_docker:
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	docker buildx create --name multi || true
	docker buildx use multi
	docker buildx inspect --bootstrap
	docker buildx build --platform linux/arm64,linux/amd64 \
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

# Create a dev environment with Shipyard and do not install the controller helm chart
create_dev_env_no_controller_no_app:
	shipyard run ./shipyard/kubernetes --var="helm_chart_install=false" --var="example_app=false"

# Create a dev environment with Shipyard and install the controller Helm chart but disable the controller to enable running it locally
create_dev_env_local_controller:
	shipyard run ./shipyard/kubernetes --var="helm_controller_enabled=false"

# Create a dev environment with Shipyard and install the controller
create_dev_env_docker_controller:
	shipyard run ./shipyard/kubernetes --var="controller_version=${VERSION}.dev"

# Create a dev environment with Shipyard and install the controller with no consul TLS or ACLs
create_dev_env_docker_controller_no_security:
	shipyard run ./shipyard/kubernetes --var="consul_acls_enabled=false" --var="consul_tls_enabled=false" --var="controller_version=${VERSION}.dev"

# Build the docusaurus documentation
build_docs:
	cd ./docs yarn install && yarn build

# Generate a new version of the Helm chart
generate_helm:
	cd ./kubernetes/controller && make manifests
	cd ./kubernetes/controller && make generate

# First generate the Helm specific kustomize config that creates the RBAC and CRDs
	kustomize build ./kubernetes/controller/config/helm -o ./deploy/kubernetes/charts/consul-release-controller/templates

# Set the version in the chart
	cp ./deploy/kubernetes/charts/consul-release-controller/Chart.tpl ./deploy/kubernetes/charts/consul-release-controller/Chart.yaml
	sedi=(-i) && [ "$(UNAME)" == "Darwin" ] && sedi=(-i '') ; \
		sed "$${sedi[@]}" -e 's/##VERSION##/${HELM_VERSION}/' ./deploy/kubernetes/charts/consul-release-controller/Chart.yaml
	
	cp ./deploy/kubernetes/charts/consul-release-controller/values.tpl ./deploy/kubernetes/charts/consul-release-controller/values.yaml
	sedi=(-i) && [ "$(UNAME)" == "Darwin" ] && sedi=(-i '') ; \
		sed "$${sedi[@]}" -e 's/##VERSION##/${VERSION}/' ./deploy/kubernetes/charts/consul-release-controller/values.yaml

# Now package the Helm chart into a tarball
	helm package ./deploy/kubernetes/charts/consul-release-controller

# Generate the index using github releases as source for binaries
	helm repo index . --merge ./docs/static/index.yaml --url=https://github.com/nicholasjackson/consul-release-controller/releases/download/v${VERSION}/
	mv ./index.yaml ./docs/static/index.yaml
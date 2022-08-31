DOCKER_REGISTRY ?= docker.io/nicholasjackson
SHELL := /bin/bash
UNAME := $(shell uname)

ifeq "$(VERSION_ENV)" ""
	VERSION=$(shell git log --pretty=format:'%h' -n 1)
	HELM_VERSION=0.0.4-dev
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
	cd functional_tests && go run . --godog.tags="@k8s_canary_existing"
	cd functional_tests && go run . --godog.tags="@k8s_canary_none"
	cd functional_tests && go run . --godog.tags="@k8s_canary_rollback"
	cd functional_tests && go run . --godog.tags="@k8s_canary_with_post_deployment_test"
	cd functional_tests && go run . --godog.tags="@k8s_canary_with_post_deployment_test_fail"

functional_tests_nomad:
	cd functional_tests && go run . --godog.tags="@nomad_canary_existing"

functional_tests_all: functional_tests_kubernetes functional_tests_nomad

# Run the functional tests, without creating the environment
# the environment can be created manually by running shipyard run ./shipyard/nomad
functional_tests_nomad_no_env:
	cd functional_tests && go run . --godog.tags="@nomad_canary_existing" --create-environment=false

# Run the functional tests, without creating the environment
# the environment can be created manually by running shipyard run ./shipyard/kubernetes
functional_tests_kubernetes_no_env:
	cd functional_tests && go run . --godog.tags="@k8s_canary_existing" --create-environment=false
	cd functional_tests && go run . --godog.tags="@k8s_canary_none" --create-environment=false
	cd functional_tests && go run . --godog.tags="@k8s_canary_rollback" --create-environment=false
	cd functional_tests && go run . --godog.tags="@k8s_canary_with_post_deployment_test" --create-environment=false
	cd functional_tests && go run . --godog.tags="@k8s_canary_with_post_deployment_test_fail" --create-environment=false

# Create a dev environment with Shipyard and do not install the controller helm chart
nomad_dev_env_local_controller:
	shipyard run ./shipyard/nomad --var="install_controller=local"

nomad_dev_env_no_controller:
	shipyard run ./shipyard/nomad --var="install_controller=docker"

# Create a dev environment with Shipyard and install the controller Helm chart but disable the controller to enable running it locally
kubernetes_dev_env_local_controller:
	shipyard run ./shipyard/kubernetes --var="helm_controller_enabled=false"

# Create a dev environment with Shipyard and install the controller
kubernetes_dev_env_docker_controller:
	shipyard run ./shipyard/kubernetes --var="controller_version=${VERSION}.dev"

# Create a dev environment with Shipyard and install the controller with no consul TLS or ACLs
kubernetes_dev_env_docker_controller_no_security:
	shipyard run ./shipyard/kubernetes --var="consul_acls_enabled=false" --var="consul_tls_enabled=false" --var="controller_version=${VERSION}.dev"

# Build the docusaurus documentation
build_docs:
	cd ./docs yarn install && yarn build

# Generate a new version of the Helm chart
generate_helm:
	cd ./pkg/controllers/kubernetes && make manifests
	cd ./pkg/controllers/kubernetes && make generate

# First generate the Helm specific kustomize config that creates the RBAC and CRDs
	kustomize build ./pkg/controllers/kubernetes/config/helm -o ./deploy/kubernetes/charts/consul-release-controller/templates

# Move the crds to the crds folder for helm
	mv ./deploy/kubernetes/charts/consul-release-controller/templates/apiextensions.k8s.io_v1_customresourcedefinition_releases.consul-release-controller.nicholasjackson.io.yaml \
		./deploy/kubernetes/charts/consul-release-controller/templates/crds/apiextensions.k8s.io_v1_customresourcedefinition_releases.consul-release-controller.nicholasjackson.io.yaml

# Set the version in the chart
	cp ./deploy/kubernetes/charts/consul-release-controller/Chart.tpl ./deploy/kubernetes/charts/consul-release-controller/Chart.yaml
	sedi=(-i) && [ "$(UNAME)" == "Darwin" ] && sedi=(-i '') ; \
		sed "$${sedi[@]}" -e 's/##VERSION##/${HELM_VERSION}/' ./deploy/kubernetes/charts/consul-release-controller/Chart.yaml

	cp ./deploy/kubernetes/charts/consul-release-controller/values.tpl ./deploy/kubernetes/charts/consul-release-controller/values.yaml
	sedi=(-i) && [ "$(UNAME)" == "Darwin" ] && sedi=(-i '') ; \
		sed "$${sedi[@]}" -e 's/##VERSION##/${VERSION}/' ./deploy/kubernetes/charts/consul-release-controller/values.yaml

# Fetch the chart deps
# helm dep up ./deploy/kubernetes/charts/consul-release-controller

# Now package the Helm chart into a tarball
	helm package ./deploy/kubernetes/charts/consul-release-controller

# Generate the index using github releases as source for binaries
	helm repo index . --merge ./docs/static/index.yaml --url=https://github.com/nicholasjackson/consul-release-controller/releases/download/v${VERSION}/
	mv ./index.yaml ./docs/static/index.yaml

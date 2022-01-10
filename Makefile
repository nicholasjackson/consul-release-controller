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
	cd functional_tests && go run main.go

# Run the functional tests, without creating the environment
# the environment can be created manually by running shipyard run ./shipyard/kubernetes
functional_tests_kubernetes_no_env:
	cd functional_tests && go run main.go --create-environment=false

# Create a new release for kubernetes
deploy_kubernetes_relase:
	curl -k https://localhost:9443/v1/releases -XPOST -d @./example/kubernetes/canary/api.json

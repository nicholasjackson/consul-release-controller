fetch_kubernetes_certs:
	mkdir -p /tmp/k8s-webhook-server/serving-certs/
	
	kubectl get secret consul-canary-webhook-certificate -n consul -o json | \
		jq -r '.data."tls.crt"' | \
		base64 -d > /tmp/k8s-webhook-server/serving-certs/tls.crt
	
	kubectl get secret consul-canary-webhook-certificate -n consul -o json | \
		jq -r '.data."tls.key"' | \
		base64 -d > /tmp/k8s-webhook-server/serving-certs/tls.key

run_kubernetes: fetch_kubernetes_certs
	go run main.go
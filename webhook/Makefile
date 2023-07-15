.DEFAULT_GOAL := help
SHELL=/bin/bash -eo pipefail

.PHONY: lint
lint: ## Run golangci-lint
	@docker run --rm -v $(shell pwd):/app \
	-w /app golangci/golangci-lint golangci-lint run -v --timeout 5m

.PHONY: build
build: ## Build webhook
	docker build -t demo-webhook:latest .
	kind load docker-image demo-webhook:latest

.PHONY: tls
tls: ## Generate webhook TLS certificates
	@cd manifests && \
	openssl genrsa -out ca.key 2048 && \
	openssl req -new -x509 -days 365 -key ca.key -subj "/C=UA/CN=demo-webhook" -out ca.crt && \
	openssl req -newkey rsa:2048 -nodes -keyout server.key -subj "/C=UA/CN=demo-webhook" -out server.csr && \
	\
	printf "subjectAltName=DNS:demo-webhook.default.svc" > extension.conf && \
	openssl x509 -req -extfile extension.conf \
	-days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt && \
	\
	kubectl create secret tls demo-webhook-tls --cert=server.crt --key=server.key \
	--dry-run=client -o yaml > webhook.tls.secret.yaml && \
	\
	base64 -w 0 ca.crt > ca.crt.b64 && \
	yq '.webhooks[].clientConfig.caBundle |= load("ca.crt.b64")' validating.config.yaml -i && \
	yq '.webhooks[].clientConfig.caBundle |= load("ca.crt.b64")' mutating.config.yaml -i && \
	\
	rm -f ca.{crt,key,crt.b64} server.{crt,csr,key} extension.conf

.PHONY: deploy
deploy: tls ## Deploy webhook
	kubectl delete --ignore-not-found=true -f manifests/
	kubectl apply -f manifests/

.PHONY: pod1
pod1: ns-create ## Pod creation should fail because of validating webhook
	kubectl run pod1 --image=nginx -n apps || :

.PHONY: pod2
pod2: ns-create ## Pod creation should succeed webhook validation
	kubectl run pod2 --image=nginx -n apps --labels=appid=12345

.PHONY: ns-create
ns-create: ns-delete ## Create test namespace (apps) with proper label (demo-webhook=enabled)
	@kubectl create ns apps 2>/dev/null || :
	@kubectl label ns apps demo-webhook=enabled --overwrite=true

.PHONY: ns-delete
ns-delete: ## Remove test namespace (apps)
	kubectl delete namespace apps --ignore-not-found=true

.PHONY: logs
logs: ## Show webhook logs
	kubectl logs -l app=demo-webhook

.PHONY: explain
explain: ## Explain webhook configuration
	kubectl explain MutatingWebhookConfiguration.webhooks --recursive
	kubectl explain MutatingWebhookConfiguration.webhooks

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-z.A-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-12s\033[0m %s\n", $$1, $$2}'

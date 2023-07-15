.DEFAULT_GOAL := help
SHELL=/bin/bash -eo pipefail

.PHONY: lint
lint: ## Run golangci-lint
	@docker run --rm -v $(shell pwd):/app \
	-w /app golangci/golangci-lint golangci-lint run -v --timeout 5m

.PHONY: build
build: ## Build controller
	docker build -t demo-controller:latest .
	kind load docker-image demo-controller:latest

.PHONY: deploy
deploy: ## Deploy controller
	kubectl delete --ignore-not-found=true -f manifests/
	kubectl apply -f manifests/

.PHONY: test-deploy
test-deploy: ns-create ## Create test deployment with 1 replica
	kubectl create deployment test-deploy -n apps --image=nginx && \
	kubectl patch deployment test-deploy -n apps \
		--patch='{"spec":{"template":{"metadata":{"labels":{"appid":"777"}}}}}'

.PHONY: test-annotate
test-annotate: ## Annotate test deployment with HA annotation
	kubectl annotate --overwrite deploy test-deploy -n apps demo-controller.local/ha=true

.PHONY: ns-create
ns-create: ns-delete ## Create test namespace (apps) with proper label (demo-webhook=enabled)
	@kubectl create ns apps 2>/dev/null || :
	@kubectl label ns apps demo-webhook=disabled --overwrite=true

.PHONY: ns-delete
ns-delete: ## Remove test namespace (apps)
	kubectl delete namespace apps --ignore-not-found=true

.PHONY: logs
logs: ## Show controller logs
	kubectl logs -l app=demo-controller

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-z.A-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-14s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
SHELL=/bin/bash -eo pipefail

.PHONY: cluster
cluster: ## Create kind cluster
	kind create cluster --config kind.yaml

.PHONY: delete
delete: ## Delete kind cluster
	kind delete cluster

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-z.A-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-10s\033[0m %s\n", $$1, $$2}'


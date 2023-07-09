# Go Kubernetes Demo
The purpose of this is to show how simple webhooks/controllers/operators might be and ease of their creation. 

## Installation
This project can fully run locally and includes automation to deploy a local Kubernetes cluster (using Kind).

### Requirements
* Container runtime (e.g. Docker)
* Kind
* kubectl
* Go
* Make
* yq

### Create Cluster
```
# First, we need to create a Kubernetes cluster:
❯ make cluster

# Make sure that the Kubernetes node is up:
❯ kubectl get nodes

# Make sure all pods are up:
❯ kubectl get pods -n kube-system
```

### Usage
```
❯ cd webhook  # OR operator/controller
❯ make build && make deploy
```

## Webhook
Simple webserver that analyse incoming json objects and does next:
- *mutating*: returns json patches (mutations)
- *validating*: returns "YES/NO" and reason

## Controller
A program that uses the same control loop technique as in electronics. Usually watches for some Kubernetes objects that could be additionally filtered and does its logic.

## Operator
Mostly the same as Controller but has its own CRD[s] (Custom Resource Definitions) with a defined field's structure. Watches for their changes and applies the desired configuration.

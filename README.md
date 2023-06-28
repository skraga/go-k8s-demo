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

### Create Cluster
First, we need to create a Kubernetes cluster:
```
❯ make cluster
kind create cluster --config kind.yaml
Creating cluster "kind" ...
 ✓ Ensuring node image (kindest/node:v1.27.0) 🖼 
 ✓ Preparing nodes 📦  
 ✓ Writing configuration 📜 
 ✓ Starting control-plane 🕹️ 
 ✓ Installing CNI 🔌 
 ✓ Installing StorageClass 💾 
Set kubectl context to "kind-kind"
You can now use your cluster with:

kubectl cluster-info --context kind-kind

Have a nice day! 👋
```

Make sure that the Kubernetes node is ready:
```
❯ kubectl get nodes
NAME                 STATUS   ROLES           AGE   VERSION
kind-control-plane   Ready    control-plane   33s   v1.27.0
```

And that system pods are running happily:
```
❯ kubectl get pods -n kube-system
NAME                                         READY   STATUS    RESTARTS   AGE
coredns-5d78c9869d-s4wz7                     1/1     Running   0          38s
coredns-5d78c9869d-xn2mt                     1/1     Running   0          38s
etcd-kind-control-plane                      1/1     Running   0          52s
kindnet-s8f9g                                1/1     Running   0          39s
kube-apiserver-kind-control-plane            1/1     Running   0          52s
kube-controller-manager-kind-control-plane   1/1     Running   0          52s
kube-proxy-wb6m6                             1/1     Running   0          39s
kube-scheduler-kind-control-plane            1/1     Running   0          52s
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

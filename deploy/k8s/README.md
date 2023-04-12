# Reorg Monitor Kubernetes Deployment

### Dependencies
- [docker](https://www.docker.com/products/docker-desktop/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/)
- [kind](https://kind.sigs.k8s.io/) (Kubernetes in Docker)
- [make](https://www.gnu.org/software/make/)

## [Development] Running Locally

#### [Prerequisite] Initialize kind cluster
```shell
kind create cluster
```

1. Build docker image and load into kind cluster
```shell
DOCKER_TAG=flashbots/reorg-monitor:0.1.0 make docker-image

kind load docker-image flashbots/reorg-monitor:0.1.0
```
2. [OPTIONAL] Navigate to development directory and inspect kubernetes manifests to ensure they are valid
```shell
cd reorg-monitor/deploy/k8s/dev

kubectl kustomize . 
```
3. [OPTIONAL] Create namespace if it doesn't exist 
```shell
# NOTE: Be sure to match the namespace name with value defined in manifests or kustomization file
kubectl create namespace reorg-monitor
```
4. Deploy services to namespace using relevant manifests
```shell
kubectl apply -k deploy/k8s/dev
```
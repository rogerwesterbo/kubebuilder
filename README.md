# kubebuilder

Showing how to build kubebuilder kubernetes operator

# Prerequisites

- A kubernetes cluster (locally or from a cloud service)
  - Kind (https://kind.sigs.k8s.io/)
  - K3d ()https://k3d.io/
  - Microk8s (https://microk8s.io/)
  - Minikube (https://minikube.sigs.k8s.io/)
- Golang (https://go.dev)
  - Other sdks: https://kubernetes.io/docs/reference/using-api/client-libraries/
- Kubebuilder (https://kubebuilder.io)

# Initialize kubebuilder project

`kubebuilder init --domain <your domain> --repo <your domain>/<module-name>`

ex:

`kubebuilder init --domain a-cool-name.io --repo a-cool-name.io/k8s`

## Add a kubebuilder api with custom resource definition (CRD)

`kubebuilder create api --group task --version v1 --kind Backup`

## Build

`make`

## Create CRD file

`make manifests`

## Install CRD in cluster:

Either install with the makefile: `make install`

Or

kubectl: `kubectl apply -f ./config/crd/bases/tasks.a-cool-name.io_backups.yaml`

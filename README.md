# Description

This Project aim to create a K8s Operator that can aggregate many TLS Secrets into one single Opaque Secret with many files.
It uses the "new" Kubernetes Custom Resource Definition API and is build using the Operator SDK Framework (https://github.com/operator-framework/operator-sdk)

## Use case

### Cert-Manager + Istio

Cert manager create a TLS Secret for each certificate you request
Istio Ingress Gateway can only mount ONE secret for the SSL certificates

Using the Cert-Operator, we can use Cert-Manager to create many TLS Secrets and merge them into a single multi-file Secret that can be used by Istio Gateways.

# Usage
You need to install many Manifests to setup the Operator.
For the moment it creates a `cert-merge` Namespace and setup the operator inside.
You can then put the `CertMerge` Manifests anywhere in the cluster.

## Operator
```
kubectl apply -f deploy/namespace.yaml
kubectl -n cert-merge apply -f deploy/service_account.yaml
kubectl -n cert-merge apply -f deploy/role.yaml
kubectl -n cert-merge apply -f deploy/role_binding.yaml
kubectl -n cert-merge apply -f deploy/certmerge_v1alpha1_certmerge_crd.yaml
kubectl -n cert-merge apply -f deploy/operator.yaml
```

## Test
Secrets are installed in the Default Namespace for testing purpose :
```
kubectl -n default apply -f deploy/test_secrets
kubectl -n default apply -f deploy/test_cr
```
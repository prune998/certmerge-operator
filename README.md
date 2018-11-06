# Description

This Project aim to create a K8s Operator that can aggregate many TLS Secrets into one single Opaque Secret with many files.
It uses the "new" Kubernetes Custom Resource Definition API and is build using the [Operator SDK Framework](https://github.com/operator-framework/operator-sdk)

## Use case

### Cert-Manager + Istio

Cert manager create a TLS Secret for each certificate you request
Istio Ingress Gateway can only mount ONE secret for the SSL certificates

Using the Cert-Operator, we can use Cert-Manager to create many TLS Secrets and merge them into a single multi-file Secret that can be used by Istio Gateways.

# Usage
You need to install many Manifests to setup the Operator.
For the moment it creates a `cert-merge` Namespace and setup the operator inside.
You can then put the `CertMerge` Manifests anywhere in the cluster.

## Building from source
You need Operator SDK CLI to be able to build.
Follow the [QuickStart](https://github.com/operator-framework/operator-sdk#quick-start) to build it.

Then : 
```
mkdir -p $GOPATH/src/github.com/prune998
cd $GOPATH/src/github.com/prune998
git clone https://github.com/prune998/certmerge-operator.git
cd certmerge-operator

operator-sdk generate k8s
operator-sdk build prune/cert-operator
docker push  prune/cert-operator
```
As you obviously can't write to the Docker Hub Registry ``prune/cert-operator``, don't `docker push` it  :)

## Using Docker Image
A docker image is available using Docker Hub Registry. Simply pull from there :
```
docker pull prune/cert-operator
```

## Deploying the Operator
You need a working Kubernetes Cluster and a configured `kubectl`.
Use `kubectl cluster-info` to ensure everything is fine prior to deploying the Operator.

The Operator is configured to deploy in `cert-operator` Namespace and use a ClusterRole to give read access to `Secret` resources.
```
kubectl apply -f deploy/namespace.yaml
kubectl -n cert-merge apply -f deploy/service_account.yaml
kubectl -n cert-merge apply -f deploy/role.yaml
kubectl -n cert-merge apply -f deploy/role_binding.yaml
kubectl -n cert-merge apply -f deploy/certmerge_v1alpha1_certmerge_crd.yaml
kubectl -n cert-merge apply -f deploy/operator.yaml
```

## Test
You can install some secrets that will trigger the test Custom Resources. Secrets are installed in the Default Namespace for testing purpose :
```
kubectl -n default apply -f deploy/test_secrets
kubectl -n default apply -f deploy/test_cr
```

# TODO
For the moment the operator is able to merge any secret from any namespace. As data inside the CertMerge's Secrets is not namespaced, you could end with one secret overwriting another one with the same name from another namespace.
The solution is to name each secret's data by `<namespace>-<secret-name>.(key|crt)`. 
I still need to check the implication when using the certificate with Istio.

# API
## CertMerge Custom Resource Definition 

```

apiVersion: certmerge.lecentre.net/v1alpha1
kind: CertMerge
metadata:
  name: "test-certmerge-labels"
spec:
  selector:
    - labelselector:
        matchLabels:
          env: "dev"
          certmerge: "true"
      namespace: default
  secretlist:
  - name: test-ingressgateway-certs
    namespace: default
  - name: test-tls-secret
    namespace: default
  name: test-cert-merge
  namespace: default
```
### Selector
The selector is a list of LabelSelectors (matchLabels only). All labels are evaluated as a boolean AND. 
ex :
1. get all `dev` secrets that are managed by `certmerge` in namespace `default`: 
    ```
      selector:
        - labelselector:
            matchLabels:
              env: "dev"
              certmerge: "true"
          namespace: default
    ```

1. get all `dev` secrets that are managed by `certmerge` in namespace `default` and `prod`: 
    ```
      selector:
        - labelselector:
            matchLabels:
              env: "dev"
              certmerge: "true"
          namespace: default
        - labelselector:
            matchLabels:
              env: "dev"
              certmerge: "true"
          namespace: prod
    ```

1. get all `dev` and `prod` secrets that are managed by `certmerge` in namespace `default`: 
    ```
      selector:
        - labelselector:
            matchLabels:
              env: "dev"
              certmerge: "true"
          namespace: default
        - labelselector:
            matchLabels:
              env: "prod"
              certmerge: "true"
          namespace: default
    ```

### Secret List
You can specify specific secrets to merge by listing them :
```
  secretlist:
  - name: test-secret-1
    namespace: default
  - name: test-secret-2
    namespace: prod
```
# Development

## KCM Installation

Follow the [KCM development guide](https://github.com/k0rdent/kcm/blob/main/docs/dev.md) or run the following commands:

  ```bash
  git clone https://github.com/k0rdent/kcm.git
  cd kcm
  make cli-install
  make dev-apply
  ```

## k0rdent-istio installation

1. Fork the [k0rdent/istio](https://github.com/k0rdent/istio) repository to your own account, e.g. `https://github.com/YOUR_USERNAME/istio`.
2. Run the following commands:

  ```bash
  cd ..
  git clone git@github.com:YOUR_USERNAME/istio.git
  cd istio

  make cli-install
  make registry-deploy
  make helm-push
  ```

3. Deploy the `k0rdent-istio` chart to your local management cluster:

  ```bash
  make dev-istio-deploy
  ```

## Deployment to AWS

1. Export your AWS credentials as described in the [KCM development guide for AWS](https://github.com/k0rdent/kcm/blob/main/docs/dev.md#aws-provider-setup)

  ```bash
  export AWS_ACCESS_KEY_ID=...
  export AWS_SECRET_ACCESS_KEY=...
  ```

2. Apply the AWS development credentials and deploy the demo cluster:

  ```bash
  cd ../kcm
  make dev-creds-apply
  cd ../istio
  ```

3. Update the cluster name and apply the Istio ClusterDeployment manifest from the demo:

  ```bash
  kubectl apply -f demo/cluster/aws-standalone-istio-cluster.yaml
  ```

The `MultiClusterService` will automatically install the `k0rdent-istio` chart to the target cluster.

## Adopted local kind cluster

This method does not help when you need a real cluster, but may help with other cases.

* Create an adopted kind cluster for quick dev/test iterations:

  ```bash
  make dev-adopted-deploy KIND_CLUSTER_NAME=adopted-cluster
  ```

* Run kind cloud provider to support external IP allocation for ingress services

  ```bash
  docker run --rm --network kind -v /var/run/docker.sock:/var/run/docker.sock registry.k8s.io/cloud-provider-kind/cloud-controller-manager:v0.7.0
  ```

* Create an adopted cluster deployment either with or without Istio:

  ```bash
  kubectl apply -f demo/clusters/adopted-cluster.yaml
  ```

* Inspect the adopted cluster:

  ```bash
  kubectl --context=kind-adopted-cluster get pod -A
  ```

## k0rdent-istio Uninstallation

To completely remove Istio from the cluster, run the following commands:

```bash
helm uninstall --wait -n istio-system k0rdent-istio
kubectl delete namespace istio-system --wait
kubectl get crd -o name | grep --color=never 'istio.io' | xargs kubectl delete
```

# Example

You can deploy and test the Istio mesh by following these steps:

## Install k0rdent-istio

Follow the steps from the [dev.md](./dev.md) to deploy Istio to both the management and remote clusters.

**Note**: If you install `k0rdent-istio` chart manually in a namespace other than istio-system, make sure to update the following values in `chart/k0rdent-istio/values.yaml`:

```yaml
global:
  caAddress: cert-manager-istio-csr.<YOUR_NAMESPACE>.svc:443
cert-manager-istio-csr:
  app:
    tls:
      certificateDNSNames:
        - cert-manager-istio-csr.<YOUR_NAMESPACE>.svc
```

## Deploy resources to the remote cluster

Create a namespace and enable automatic sidecar injection for all pods in that namespace:

```bash
kubectl create namespace sample
kubectl label namespace sample istio-injection=enabled
```

Deploy the `helloworld` service and application to test the Istio connection:

```bash
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.27/samples/helloworld/helloworld.yaml -l service=helloworld -n sample
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.27/samples/helloworld/helloworld.yaml -l version=v1 -n sample
```

## Connection testing

In the management cluster, create a namespace and enable sidecar injection:

```bash
kubectl create namespace sample
kubectl label namespace sample istio-injection=enabled
```

Deploy the `sleep` pod to use it for testing connectivity:

```bash
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.27/samples/sleep/sleep.yaml -n sample
```

Run a curl command from the sleep pod to reach the service in the remote cluster:

```bash
kubectl exec -n sample -c sleep "$(kubectl get pod -n sample -l app=sleep -o jsonpath='{.items[0].metadata.name}')" -- curl -sS helloworld.sample:5000/hello
```

If the connection is successful, you should see an output similar to:

```planetext
Hello version: v2, instance: helloworld-v2-6746879bdd-55v7z
```

This indicates that the connection has been successfully established and that the management cluster can access the remote cluster.

## Troubleshooting

If the `curl` command returns an error, it means the connection was not established correctly. Follow these steps to debug and fix the issue:

### Missing Istio sidecar

The pod you’re using to run `curl` must include the `istio-proxy` container. Ensure that sidecar injection is enabled either for the specific pod or for the entire namespace.

### Remote secret not propagated

In some cases, the connection might fail due to a missing remote secret. Check the logs of the `k0rdent-istio-operator`, which is responsible for creating the remote secret. The secret should be automatically distributed to the remote cluster via a MultiClusterService. Make sure the same secret exists in the remote cluster.

### Useful debug tool

#### Kiali

To visualize the mesh and identify potential issues, install [`Kiali`](https://istio.io/latest/docs/ops/integrations/kiali/#installation) and [Prometheus](https://istio.io/latest/docs/ops/integrations/prometheus/) in the management cluster.

Run Kiali locally using:

```bash
kubectl port-forward -n istio-system svc/kiali 20001:20001
```

#### Istioctl

You can also debug the Istio mesh using the istioctl tool. Install it by following the [official guide](https://istio.io/latest/docs/ops/diagnostic-tools/istioctl/#install-hahahugoshortcode1244s2hbhb).

For example, to check if the management cluster is synchronized with remote clusters, run:

```bash
istioctl remote-clusters
```

For more available commands, see the [istioctl reference](https://istio.io/latest/docs/reference/commands/istioctl/).

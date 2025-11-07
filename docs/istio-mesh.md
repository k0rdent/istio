# Istio Mesh

`k0rdent.mirantis.com/istio-mesh` label isolates remote secret propagation across clusters. Without it, remote secrets are propagated to all clusters using the large default mesh.

## How to Enable it

To create an Istio Mesh, use this label on the `ClusterDeployment` that you want to isolate within a specific mesh:

```yaml
k0rdent.mirantis.com/istio-mesh: <ISTIO_MESH_NAME>
```

Set this label with the same key (`k0rdent.mirantis.com/istio-mesh`) and the same value (`<ISTIO_MESH_NAME>`) on every cluster that you want to include in the same mesh.

## KCM Region

If you want to use KCM Region with Istio, apply the label `k0rdent.mirantis.com/istio-mesh: <ISTIO_MESH_NAME>` to every `ClusterDeployment` resource in the KCM Region. Additionally, apply the same label to the `Region` resource itself to ensure that connectivity is isolated across all clusters within that region.

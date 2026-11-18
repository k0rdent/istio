# Istio Mesh

`k0rdent.mirantis.com/istio-mesh` label isolates remote secret propagation across clusters. It is a required label.

## How to Enable it

To create an Istio Mesh, use this label on the `ClusterDeployment` that you want to isolate within a specific mesh:

```yaml
k0rdent.mirantis.com/istio-mesh: <ISTIO_MESH_NAME>
```

Set this label with the same key (`k0rdent.mirantis.com/istio-mesh`) and the same value (`<ISTIO_MESH_NAME>`) on every cluster that you want to include in the same mesh.

## Clusters Without the Label

A cluster without the `k0rdent.mirantis.com/istio-mesh` label is isolated: its remote secret is delivered only to the cluster itself and is never shared with other clusters. The operator emits an `IstioMeshLabelMissing` warning event on such `ClusterDeployment` objects.

> [!WARNING]
> Previous versions shared remote secrets between all clusters without the label. After upgrading, clusters that relied on this implicit shared mesh must be labelled with a common `k0rdent.mirantis.com/istio-mesh` value to keep cross-cluster connectivity.

## Changing the Label on Existing Clusters

The label can be added, changed or removed on an existing `ClusterDeployment`. The operator updates the remote secret propagation `MultiClusterService` accordingly, and remote secrets are withdrawn from clusters that no longer match.

Remote secrets contain credentials for the `istio-reader-service-account`. If they were already distributed to clusters outside the intended mesh, rotate the service account token on the affected clusters.

## Management Cluster

When `managementCluster.includeInMesh` is enabled, set `managementCluster.istioMesh` in the `k0rdent-istio` chart values to add the management cluster to a mesh. Without it, the management cluster remote secret is not shared with other clusters.

## KCM Region

If you want to use KCM Region with Istio, apply the label `k0rdent.mirantis.com/istio-mesh: <ISTIO_MESH_NAME>` to every `ClusterDeployment` resource in the KCM Region. Additionally, apply the same label to the `Region` resource itself to ensure that connectivity is isolated across all clusters within that region.

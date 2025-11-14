# k0rdent Istio

k0rdent Istio simplifies the deployment and management of Istio across management and remote clusters, providing automatic inter-cluster connectivity and certificate management.

## Multicluster Architecture

The reference architecture for this project is based on the [Multi-Primary on Different Networks](https://istio.io/latest/docs/setup/install/multicluster/multi-primary_multi-network) topology.

### Components Deployed

k0rdent Istio automates the setup of all key components required for a secure multi-cluster service mesh:

* **Self-signed Root CA** — deployed on the management cluster.
  See the [reference architecture](https://istio.io/latest/docs/tasks/security/cert-management/plugin-ca-cert/).
* **Intermediate CA** — automatically generated for each Istio cluster (labeled with `k0rdent.mirantis.com/istio-role: member`) by the k0rdent-istio-operator.
* **Remote secrets** — created for each Istio cluster by the k0rdent-istio-operator to enable cross-cluster endpoints discovery.
* **Istio Gateway** — deployed in each cluster to provide secure inter-cluster communication protected by [mTLS](https://istio.io/latest/docs/tasks/security/authentication/authn-policy/#enable-mutual-tls-per-workload)

This architecture ensures automatic mesh connectivity, consistent CA hierarchy, and secure cross-cluster communication without manual configuration.

## Charts Overview

* k0rdent-istio — deploys the full Istio system and manages mesh connectivity. It can be installed on both management and remote clusters and is automatically applied by the MultiClusterService to any cluster labeled `k0rdent.mirantis.com/istio-role: member`.

## CONTRIBUTION

* Follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification

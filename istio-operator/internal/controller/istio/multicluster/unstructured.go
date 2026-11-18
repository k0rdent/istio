package multicluster

import (
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// toUnstructured converts the MultiClusterService to an unstructured object, dropping
// an empty spec.serviceSpec.provider. The KCM API version used by the operator always
// serializes the provider struct, while newer KCM versions reject an empty provider.
func toUnstructured(mcs *kcmv1beta1.MultiClusterService) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(mcs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert MultiClusterService to unstructured: %w", err)
	}

	u := &unstructured.Unstructured{Object: obj}
	u.SetGroupVersionKind(kcmv1beta1.GroupVersion.WithKind(kcmv1beta1.MultiClusterServiceKind))

	provider, found, err := unstructured.NestedMap(obj, "spec", "serviceSpec", "provider")
	if err == nil && found && len(provider) == 0 {
		unstructured.RemoveNestedField(obj, "spec", "serviceSpec", "provider")
	}
	unstructured.RemoveNestedField(obj, "status")

	return u, nil
}

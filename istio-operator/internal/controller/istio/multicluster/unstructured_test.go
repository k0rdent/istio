package multicluster

import (
	"testing"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestToUnstructured_DropsEmptyProvider(t *testing.T) {
	setupTest(t, "1.0.0")
	mcs := New(nil).generateMultiClusterService(newClusterDeployment())

	obj, err := toUnstructured(mcs)
	if err != nil {
		t.Fatalf("toUnstructured returned error: %v", err)
	}

	if _, found, _ := unstructured.NestedFieldNoCopy(obj.Object, "spec", "serviceSpec", "provider"); found {
		t.Errorf("expected empty spec.serviceSpec.provider to be dropped")
	}
	if _, found, _ := unstructured.NestedFieldNoCopy(obj.Object, "spec", "serviceSpec", "services"); !found {
		t.Errorf("expected spec.serviceSpec.services to be kept")
	}
	if gvk := obj.GroupVersionKind(); gvk.Kind != kcmv1beta1.MultiClusterServiceKind || gvk.GroupVersion() != kcmv1beta1.GroupVersion {
		t.Errorf("unexpected GVK: %v", gvk)
	}
}

func TestToUnstructured_KeepsNonEmptyProvider(t *testing.T) {
	setupTest(t, "1.0.0")
	mcs := New(nil).generateMultiClusterService(newClusterDeployment())
	mcs.Spec.ServiceSpec.Provider.Name = "custom-provider"

	obj, err := toUnstructured(mcs)
	if err != nil {
		t.Fatalf("toUnstructured returned error: %v", err)
	}

	name, found, _ := unstructured.NestedString(obj.Object, "spec", "serviceSpec", "provider", "name")
	if !found || name != "custom-provider" {
		t.Errorf("expected provider name %q, got %q (found=%v)", "custom-provider", name, found)
	}
}

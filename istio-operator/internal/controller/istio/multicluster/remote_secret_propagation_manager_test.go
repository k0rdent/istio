package multicluster

import (
	"context"
	"strings"
	"testing"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/istio/istio-operator/internal/controller/istio"
	"github.com/k0rdent/istio/istio-operator/internal/controller/record"
	"github.com/k0rdent/istio/istio-operator/internal/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTryCreate_CreatesMCSWhenNotExists(t *testing.T) {
	setupTest(t, "1.0.0")

	cd := newClusterDeployment()
	c := newFakeClient(t)
	manager := New(c)

	if err := manager.TryCreate(context.Background(), cd); err != nil {
		t.Fatalf("TryCreate returned error: %v", err)
	}

	mcs := &kcmv1beta1.MultiClusterService{}
	if err := c.Get(context.Background(), types.NamespacedName{
		Name: MultiClusterServiceName(cd.Name, cd.Namespace),
	}, mcs); err != nil {
		t.Fatalf("MCS was not created: %v", err)
	}

	if got := mcs.Labels[labels.K0rdentIstioVersionLabel]; got != "1.0.0" {
		t.Errorf("expected version label %q, got %q", "1.0.0", got)
	}
}

func TestTryCreate_UpdatesMCSWhenVersionChanges(t *testing.T) {
	setupTest(t, "1.0.0")

	cd := newClusterDeployment()

	// Pre-create MCS with old version label.
	existingMCS := &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name: MultiClusterServiceName(cd.Name, cd.Namespace),
			Labels: map[string]string{
				labels.K0rdentIstioVersionLabel: "1.0.0",
			},
		},
	}
	c := newFakeClient(t, existingMCS)
	manager := New(c)

	// Simulate a version upgrade.
	istio.ReleaseVersion = "1.1.0"

	if err := manager.TryCreate(context.Background(), cd); err != nil {
		t.Fatalf("TryCreate returned error: %v", err)
	}

	mcs := &kcmv1beta1.MultiClusterService{}
	if err := c.Get(context.Background(), types.NamespacedName{
		Name: MultiClusterServiceName(cd.Name, cd.Namespace),
	}, mcs); err != nil {
		t.Fatalf("failed to get MCS: %v", err)
	}

	if got := mcs.Labels[labels.K0rdentIstioVersionLabel]; got != "1.1.0" {
		t.Errorf("expected updated version label %q, got %q", "1.1.0", got)
	}
}

func TestTryCreate_SkipsUpdateWhenVersionUnchanged(t *testing.T) {
	setupTest(t, "1.0.0")

	cd := newClusterDeployment()

	existingMCS := New(nil).generateMultiClusterService(cd)
	existingMCS.ResourceVersion = "999"
	c := newFakeClient(t, existingMCS)
	manager := New(c)

	if err := manager.TryCreate(context.Background(), cd); err != nil {
		t.Fatalf("TryCreate returned error: %v", err)
	}

	mcs := &kcmv1beta1.MultiClusterService{}
	if err := c.Get(context.Background(), types.NamespacedName{
		Name: MultiClusterServiceName(cd.Name, cd.Namespace),
	}, mcs); err != nil {
		t.Fatalf("failed to get MCS: %v", err)
	}

	// ResourceVersion must not change — no update was issued.
	if mcs.ResourceVersion != "999" {
		t.Errorf("expected ResourceVersion %q (no update), got %q", "999", mcs.ResourceVersion)
	}
}

func TestTryCreate_SequentialVersionUpgrades(t *testing.T) {
	versions := []string{"1.0.0", "1.1.0", "2.0.0", "2.1.0"}

	cd := newClusterDeployment()
	c := newFakeClient(t)
	manager := New(c)

	for _, version := range versions {
		setupTest(t, version)

		if err := manager.TryCreate(context.Background(), cd); err != nil {
			t.Fatalf("TryCreate returned error at version %s: %v", version, err)
		}

		mcs := &kcmv1beta1.MultiClusterService{}
		if err := c.Get(context.Background(), types.NamespacedName{
			Name: MultiClusterServiceName(cd.Name, cd.Namespace),
		}, mcs); err != nil {
			t.Fatalf("failed to get MCS at version %s: %v", version, err)
		}

		if got := mcs.Labels[labels.K0rdentIstioVersionLabel]; got != version {
			t.Errorf("version %s: expected label %q, got %q", version, version, got)
		}
	}
}

// setupTest initialises shared global state required by the manager under test.
func setupTest(t *testing.T, releaseVersion string) {
	t.Helper()
	record.DefaultRecorder = events.NewFakeRecorder(16)
	istio.IstioSystemNamespace = "istio-system"
	istio.IstioReleaseName = "test-istio"
	istio.ReleaseVersion = releaseVersion
}

func newClusterDeployment() *kcmv1beta1.ClusterDeployment {
	return &kcmv1beta1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-a",
			Namespace: "default",
			Labels: map[string]string{
				labels.IstioRoleLabel: labels.IstioRoleLabelMemberValue,
			},
		},
		Spec: kcmv1beta1.ClusterDeploymentSpec{
			Template:   "test-template",
			Credential: "test-credential",
		},
	}
}

func newFakeClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()

	s := runtime.NewScheme()
	if err := k8sscheme.AddToScheme(s); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}
	if err := kcmv1beta1.AddToScheme(s); err != nil {
		t.Fatalf("failed to add kcm scheme: %v", err)
	}

	return fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(objects...).
		Build()
}

func getMCS(t *testing.T, c client.Client, cd *kcmv1beta1.ClusterDeployment) *kcmv1beta1.MultiClusterService {
	t.Helper()
	mcs := &kcmv1beta1.MultiClusterService{}
	if err := c.Get(context.Background(), types.NamespacedName{
		Name: MultiClusterServiceName(cd.Name, cd.Namespace),
	}, mcs); err != nil {
		t.Fatalf("failed to get MCS: %v", err)
	}
	return mcs
}

func remoteSecretValues(t *testing.T, mcs *kcmv1beta1.MultiClusterService) string {
	t.Helper()
	if len(mcs.Spec.ServiceSpec.Services) == 0 {
		t.Fatalf("MCS has no services")
	}
	return mcs.Spec.ServiceSpec.Services[0].Values
}

func hasDoesNotExistMeshExpression(mcs *kcmv1beta1.MultiClusterService) bool {
	for _, e := range mcs.Spec.ClusterSelector.MatchExpressions {
		if e.Key == labels.IstioMeshLabel && e.Operator == metav1.LabelSelectorOpDoesNotExist {
			return true
		}
	}
	return false
}

func TestTryCreate_NoMeshLabelScopesRemoteSecretToSelf(t *testing.T) {
	setupTest(t, "1.0.0")
	cd := newClusterDeployment()
	c := newFakeClient(t)

	if err := New(c).TryCreate(context.Background(), cd); err != nil {
		t.Fatalf("TryCreate returned error: %v", err)
	}

	mcs := getMCS(t, c, cd)
	if !hasDoesNotExistMeshExpression(mcs) {
		t.Errorf("expected DoesNotExist mesh selector, got %+v", mcs.Spec.ClusterSelector)
	}
	values := remoteSecretValues(t, mcs)
	if !strings.Contains(values, "$eligible") || !strings.Contains(values, "adler32sum") {
		t.Errorf("expected remote secret propagation to be scoped to the cluster itself, got:\n%s", values)
	}
}

func TestTryCreate_MeshLabelPropagatesRemoteSecretToMesh(t *testing.T) {
	setupTest(t, "1.0.0")
	cd := newClusterDeployment()
	cd.Labels[labels.IstioMeshLabel] = "tenant-a"
	c := newFakeClient(t)

	if err := New(c).TryCreate(context.Background(), cd); err != nil {
		t.Fatalf("TryCreate returned error: %v", err)
	}

	mcs := getMCS(t, c, cd)
	if got := mcs.Spec.ClusterSelector.MatchLabels[labels.IstioMeshLabel]; got != "tenant-a" {
		t.Errorf("expected mesh selector %q, got %q", "tenant-a", got)
	}
	if hasDoesNotExistMeshExpression(mcs) {
		t.Errorf("unexpected DoesNotExist mesh selector")
	}
	if values := remoteSecretValues(t, mcs); strings.Contains(values, "$eligible") {
		t.Errorf("expected unscoped remote secret propagation, got:\n%s", values)
	}
}

func TestTryCreate_UpdatesMCSWhenMeshLabelChanges(t *testing.T) {
	setupTest(t, "1.0.0")
	cd := newClusterDeployment()
	c := newFakeClient(t)
	manager := New(c)

	steps := []struct {
		name string
		mesh string
	}{
		{"no mesh", ""},
		{"mesh added", "tenant-a"},
		{"mesh changed", "tenant-b"},
		{"mesh removed", ""},
	}

	for _, step := range steps {
		if step.mesh == "" {
			delete(cd.Labels, labels.IstioMeshLabel)
		} else {
			cd.Labels[labels.IstioMeshLabel] = step.mesh
		}

		if err := manager.TryCreate(context.Background(), cd); err != nil {
			t.Fatalf("%s: TryCreate returned error: %v", step.name, err)
		}

		mcs := getMCS(t, c, cd)
		got := mcs.Spec.ClusterSelector.MatchLabels[labels.IstioMeshLabel]
		if got != step.mesh {
			t.Errorf("%s: expected mesh selector %q, got %q", step.name, step.mesh, got)
		}
		if (step.mesh == "") != hasDoesNotExistMeshExpression(mcs) {
			t.Errorf("%s: unexpected DoesNotExist expression state: %+v", step.name, mcs.Spec.ClusterSelector)
		}
	}
}

func TestTryCreate_UpdatesLegacyMCSWithoutSpecHash(t *testing.T) {
	setupTest(t, "1.0.0")
	cd := newClusterDeployment()
	cd.Labels[labels.IstioMeshLabel] = "tenant-a"

	// MCS created by an older operator version: same release version, no spec hash, wide selector.
	legacy := &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name: MultiClusterServiceName(cd.Name, cd.Namespace),
			Labels: map[string]string{
				labels.K0rdentIstioVersionLabel: "1.0.0",
			},
		},
		Spec: kcmv1beta1.MultiClusterServiceSpec{
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{labels.IstioRoleLabel: labels.IstioRoleLabelMemberValue},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: labels.IstioMeshLabel, Operator: metav1.LabelSelectorOpDoesNotExist},
				},
			},
		},
	}
	c := newFakeClient(t, legacy)

	if err := New(c).TryCreate(context.Background(), cd); err != nil {
		t.Fatalf("TryCreate returned error: %v", err)
	}

	mcs := getMCS(t, c, cd)
	if got := mcs.Spec.ClusterSelector.MatchLabels[labels.IstioMeshLabel]; got != "tenant-a" {
		t.Errorf("expected mesh selector %q, got %q", "tenant-a", got)
	}
	if hasDoesNotExistMeshExpression(mcs) {
		t.Errorf("expected DoesNotExist expression to be removed")
	}
	if mcs.Annotations[SpecHashAnnotation] == "" {
		t.Errorf("expected spec hash annotation to be set")
	}
}

package k8s

import (
	"context"
	"testing"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetKubeconfigFromClusterDeploymentUsesClusterDeploymentNamespace(t *testing.T) {
	t.Parallel()

	cdNamespace := "tenant-a"
	secretName := "demo-kubeconfig"

	cd := &kcmv1beta1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: cdNamespace,
		},
		Spec: kcmv1beta1.ClusterDeploymentSpec{
			Credential: "demo",
		},
	}

	tenantSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: cdNamespace,
		},
		Data: map[string][]byte{
			"value": []byte("tenant-kubeconfig"),
		},
	}

	defaultSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: DefaultKCMSystemNamespace,
		},
		Data: map[string][]byte{
			"value": []byte("default-kubeconfig"),
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cd, tenantSecret, defaultSecret).Build()

	kubeconfig, err := GetKubeconfigFromClusterDeployment(context.Background(), c, cd)
	if err != nil {
		t.Fatalf("GetKubeconfigFromClusterDeployment returned error: %v", err)
	}

	if string(kubeconfig) != "tenant-kubeconfig" {
		t.Fatalf("expected tenant namespace kubeconfig, got %q", string(kubeconfig))
	}
}

func TestCreatedInKCMRegionUsesClusterDeploymentNamespace(t *testing.T) {
	t.Parallel()

	cd := &kcmv1beta1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "tenant-b",
		},
		Status: kcmv1beta1.ClusterDeploymentStatus{
			Region: "regional-cluster",
		},
	}

	createdInRegion := CreatedInKCMRegion(cd)

	if !createdInRegion {
		t.Fatalf("expected CreatedInKCMRegion to return true")
	}
}

func TestCreatedInKCMRegionFailsWhenCredentialMissingInClusterDeploymentNamespace(t *testing.T) {
	t.Parallel()

	cd := &kcmv1beta1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "tenant-c",
		},
	}

	createdInRegion := CreatedInKCMRegion(cd)

	if createdInRegion {
		t.Fatalf("expected CreatedInKCMRegion to return false when status region is empty")
	}
}

func TestGetKcmRegionClusterNameRelatedToClusterDeploymentPrefersClusterNamespace(t *testing.T) {
	t.Parallel()

	cd := &kcmv1beta1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "tenant-d",
		},
		Spec: kcmv1beta1.ClusterDeploymentSpec{
			Credential: "demo-credential",
		},
	}

	tenantCredential := &kcmv1beta1.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-credential",
			Namespace: "tenant-d",
		},
		Spec: kcmv1beta1.CredentialSpec{
			Region: "tenant-region",
		},
	}

	fallbackCredential := &kcmv1beta1.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-credential",
			Namespace: DefaultKCMSystemNamespace,
		},
		Spec: kcmv1beta1.CredentialSpec{
			Region: "fallback-region",
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cd, tenantCredential, fallbackCredential).Build()

	regionName, err := GetKcmRegionClusterNameRelatedToClusterDeployment(context.Background(), c, cd)
	if err != nil {
		t.Fatalf("GetKcmRegionClusterNameRelatedToClusterDeployment returned error: %v", err)
	}

	if regionName != "tenant-region" {
		t.Fatalf("expected tenant-region, got %q", regionName)
	}
}

func TestGetKcmRegionClusterNameRelatedToClusterDeploymentFailsWhenCredentialMissingInClusterDeploymentNamespace(t *testing.T) {
	t.Parallel()

	cd := &kcmv1beta1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "tenant-e",
		},
		Spec: kcmv1beta1.ClusterDeploymentSpec{
			Credential: "demo-credential",
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cd).Build()

	regionName, err := GetKcmRegionClusterNameRelatedToClusterDeployment(context.Background(), c, cd)
	if err == nil {
		t.Fatalf("expected GetKcmRegionClusterNameRelatedToClusterDeployment to fail when credential is missing in ClusterDeployment namespace")
	}

	if regionName != "" {
		t.Fatalf("expected empty region on missing credential, got %q", regionName)
	}
}

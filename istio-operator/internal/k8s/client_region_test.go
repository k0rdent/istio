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

	cdNamespace := "tenant-b"
	credentialName := "demo-credential"

	cd := &kcmv1beta1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: cdNamespace,
		},
		Spec: kcmv1beta1.ClusterDeploymentSpec{
			Credential: credentialName,
		},
	}

	tenantCredential := &kcmv1beta1.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credentialName,
			Namespace: cdNamespace,
		},
		Spec: kcmv1beta1.CredentialSpec{
			Region: "regional-cluster",
		},
	}

	defaultCredential := &kcmv1beta1.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credentialName,
			Namespace: DefaultKCMSystemNamespace,
		},
		Spec: kcmv1beta1.CredentialSpec{},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cd, tenantCredential, defaultCredential).Build()

	createdInRegion, err := CreatedInKCMRegion(context.Background(), c, cd)
	if err != nil {
		t.Fatalf("CreatedInKCMRegion returned error: %v", err)
	}

	if !createdInRegion {
		t.Fatalf("expected CreatedInKCMRegion to return true")
	}
}

func TestCreatedInKCMRegionFallsBackToDefaultNamespace(t *testing.T) {
	t.Parallel()

	cd := &kcmv1beta1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "tenant-c",
		},
		Spec: kcmv1beta1.ClusterDeploymentSpec{
			Credential: "demo-credential",
		},
	}

	fallbackCredential := &kcmv1beta1.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-credential",
			Namespace: DefaultKCMSystemNamespace,
		},
		Spec: kcmv1beta1.CredentialSpec{
			Region: "regional-cluster",
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cd, fallbackCredential).Build()

	createdInRegion, err := CreatedInKCMRegion(context.Background(), c, cd)
	if err != nil {
		t.Fatalf("CreatedInKCMRegion returned error: %v", err)
	}

	if !createdInRegion {
		t.Fatalf("expected CreatedInKCMRegion to return true for fallback credential")
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

func TestGetKcmRegionClusterNameRelatedToClusterDeploymentFallsBackToDefaultNamespace(t *testing.T) {
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

	fallbackCredential := &kcmv1beta1.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-credential",
			Namespace: DefaultKCMSystemNamespace,
		},
		Spec: kcmv1beta1.CredentialSpec{
			Region: "fallback-region",
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cd, fallbackCredential).Build()

	regionName, err := GetKcmRegionClusterNameRelatedToClusterDeployment(context.Background(), c, cd)
	if err != nil {
		t.Fatalf("GetKcmRegionClusterNameRelatedToClusterDeployment returned error: %v", err)
	}

	if regionName != "fallback-region" {
		t.Fatalf("expected fallback-region, got %q", regionName)
	}
}

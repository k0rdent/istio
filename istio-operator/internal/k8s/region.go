package k8s

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// If ClusterDeployment status has a non-empty region field,
// we assume the cluster was created in that KCM region.
func CreatedInKCMRegion(cd *kcmv1beta1.ClusterDeployment) bool {
	return cd.Status.Region != ""
}

func GetKcmRegionClusterNameRelatedToClusterDeployment(ctx context.Context, client client.Client, cd *kcmv1beta1.ClusterDeployment) (string, error) {
	cred, err := getCredentialForClusterDeployment(ctx, client, cd)
	if err != nil {
		return "", err
	}

	return cred.Spec.Region, nil
}

func getCredentialForClusterDeployment(ctx context.Context, client client.Client, cd *kcmv1beta1.ClusterDeployment) (*kcmv1beta1.Credential, error) {
	cred := new(kcmv1beta1.Credential)
	sameNamespaceName := types.NamespacedName{
		Name:      cd.Spec.Credential,
		Namespace: cd.Namespace,
	}

	if err := client.Get(ctx, sameNamespaceName, cred); err != nil {
		return nil, err
	}

	return cred, nil
}

func GetKubeconfigByRegionName(ctx context.Context, client client.Client, regionName string) ([]byte, error) {
	if regionName == "" {
		return nil, fmt.Errorf("region name is empty")
	}

	region := new(kcmv1beta1.Region)
	if err := client.Get(ctx, types.NamespacedName{Name: regionName}, region); err != nil {
		return nil, fmt.Errorf("failed to get Region %s: %v", regionName, err)
	}

	if region.Spec.ClusterDeployment != nil {
		cd := new(kcmv1beta1.ClusterDeployment)
		if err := client.Get(ctx, types.NamespacedName{
			Name:      region.Spec.ClusterDeployment.Name,
			Namespace: region.Spec.ClusterDeployment.Namespace,
		}, cd); err != nil {
			return nil, fmt.Errorf("failed to get ClusterDeployment %s: %v", region.Spec.ClusterDeployment.Name, err)
		}

		kubeconfig, err := GetKubeconfigFromClusterDeployment(ctx, client, cd)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubeconfig from ClusterDeployment: %v", err)
		}
		return kubeconfig, nil
	}

	if region.Spec.KubeConfig != nil {
		return GetKubeconfigFromSecret(ctx, client, region.Spec.KubeConfig.Name)
	}

	return nil, nil
}

package crds

import (
	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "k0rdent.mirantis.com", Version: "v1beta1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(
		&kcmv1beta1.ClusterDeployment{},
		&kcmv1beta1.ClusterDeploymentList{},
		&kcmv1beta1.MultiClusterService{},
		&kcmv1beta1.MultiClusterServiceList{},
		&Credential{},
		&CredentialList{},
		&Region{},
		&RegionList{},
	)
}

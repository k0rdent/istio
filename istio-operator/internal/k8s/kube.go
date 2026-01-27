package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/version"
	kubeVersion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
)

func GetKubernetesVersion(client *kubernetes.Clientset) (*kubeVersion.Info, error) {
	return client.Discovery().ServerVersion()
}

// IsAtLeastVersion returns true if the client is at least the specified version.
// For example, on Kubernetes v1.15.2, IsAtLeastVersion(13) == true, IsAtLeastVersion(17) == false
func IsAtLeastVersion(client *kubernetes.Clientset, minorVersion uint) bool {
	clusterVersion, err := GetKubernetesVersion(client)
	if err != nil {
		return true
	}
	return IsKubeAtLeastOrLessThanVersion(clusterVersion, minorVersion, true)
}

// IsKubeAtLeastOrLessThanVersion returns if the kubernetes version is at least or less than the specified version.
func IsKubeAtLeastOrLessThanVersion(clusterVersion *kubeVersion.Info, minorVersion uint, atLeast bool) bool {
	if clusterVersion == nil {
		return true
	}
	cv, err := version.ParseGeneric(fmt.Sprintf("v%s.%s.0", clusterVersion.Major, clusterVersion.Minor))
	if err != nil {
		return true
	}
	ev, err := version.ParseGeneric(fmt.Sprintf("v1.%d.0", minorVersion))
	if err != nil {
		return true
	}
	if atLeast {
		return cv.AtLeast(ev)
	}
	return cv.LessThan(ev)
}

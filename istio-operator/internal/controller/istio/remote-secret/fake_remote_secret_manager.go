package remotesecret

import (
	"context"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/istio/istio-operator/internal/controller/istio"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FakeRemoteSecretCreator struct{}

func NewFakeManager(c client.Client) *RemoteSecretManager {
	return &RemoteSecretManager{
		client:                    c,
		IIstioRemoteSecretCreator: NewFakeRemoteSecretCreator(),
	}
}

func NewFakeRemoteSecretCreator() IIstioRemoteSecretCreator {
	return &FakeRemoteSecretCreator{}
}

func (f *FakeRemoteSecretCreator) GetRemoteSecret(ctx context.Context, kubeconfig []byte, cd *kcmv1beta1.ClusterDeployment, opt CreateOptions) (*corev1.Secret, error) {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: istio.IstioSystemNamespace,
			Name:      GetRemoteSecretName(cd.Name, cd.Namespace),
			Labels:    map[string]string{},
		},
		StringData: map[string]string{
			"value": "Fake values",
		},
	}, nil
}

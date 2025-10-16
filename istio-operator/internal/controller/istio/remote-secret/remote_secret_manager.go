package remotesecret

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/istio/istio-operator/internal/controller/istio"
	"github.com/k0rdent/istio/istio-operator/internal/controller/record"
	"github.com/k0rdent/istio/istio-operator/internal/controller/utils"
	"github.com/k0rdent/istio/istio-operator/internal/k8s"
	"istio.io/istio/istioctl/pkg/multicluster"
	"istio.io/istio/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RemoteSecretManager struct {
	client client.Client
	IIstioRemoteSecretCreator
}

func New(c client.Client) *RemoteSecretManager {
	return &RemoteSecretManager{
		client:                    c,
		IIstioRemoteSecretCreator: NewIstioRemoteSecret(),
	}
}

// Function tries to delete the remote secret
func (rs *RemoteSecretManager) TryDelete(ctx context.Context, request ctrl.Request) error {
	log := log.FromContext(ctx)
	log.Info("Trying to delete remote secret")

	if err := rs.client.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetRemoteSecretName(request.Name, request.Namespace),
			Namespace: istio.IstioSystemNamespace,
		},
	}); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Remote secret already deleted")
			return nil
		}
		return err
	}

	rs.sendDeletionEvent(request)
	log.Info("Remote secret successfully deleted")
	return nil
}

// Function handles the creation of a remote secret
func (rs *RemoteSecretManager) TryCreate(ctx context.Context, clusterDeployment *kcmv1beta1.ClusterDeployment) error {
	log := log.FromContext(ctx)
	log.Info("Trying to create remote secret")

	if !rs.isClusterDeploymentReady(clusterDeployment) {
		log.Info("Cluster deployment is not ready")
		return nil
	}

	exists, err := rs.remoteSecretExists(ctx, clusterDeployment)
	if err != nil {
		return fmt.Errorf("failed to check remote secret: %v", err)
	}

	if exists {
		log.Info("Remote secret already exists")
		return nil
	}

	kubeconfig, err := rs.GetKubeconfigFromSecret(ctx, clusterDeployment)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig from secret: %v", err)
	}

	remoteSecret, err := rs.GetRemoteSecret(ctx, kubeconfig, clusterDeployment)
	if err != nil {
		return fmt.Errorf("failed to create remote secret: %v", err)
	}

	if err := rs.createSecretResource(ctx, remoteSecret); err != nil {
		log.Error(err, "failed to create remote secret")
		return fmt.Errorf("failed to create remote secret: %v", err)
	}

	rs.sendCreationEvent(clusterDeployment)
	log.Info("Remote secret successfully created")
	return nil
}

// Function retrieves and decodes a kubeconfig from a Secret
func (rs *RemoteSecretManager) GetKubeconfigFromSecret(ctx context.Context, clusterDeployment *kcmv1beta1.ClusterDeployment) ([]byte, error) {
	log := log.FromContext(ctx)
	secretFullName := rs.getFullSecretName(clusterDeployment)

	secret, err := k8s.GetSecret(ctx, rs.client, secretFullName, clusterDeployment.Namespace)
	if err != nil {
		log.Error(err, fmt.Sprintf("Unable to fetch Secret '%s'", secretFullName))
		return nil, err
	}

	log.Info("Secret found", "name", secret.Name, "namespace", secret.Namespace)

	kubeconfigRaw := k8s.GetSecretValue(secret)
	if kubeconfigRaw == nil {
		return nil, fmt.Errorf("kubeconfig secret does not contain 'value' key")
	}
	return kubeconfigRaw, nil
}

// Function checks if the cluster deployment is in a ready state
func (rs *RemoteSecretManager) isClusterDeploymentReady(cd *kcmv1beta1.ClusterDeployment) bool {
	readiness := false

	for _, condition := range *cd.GetConditions() {
		if utils.IsAdopted(cd) {
			if condition.Type == kcmv1beta1.ReadyCondition {
				readiness = true
			}
		} else {
			if condition.Type == kcmv1beta1.CAPIClusterSummaryCondition {
				readiness = true
			}
		}
	}

	return readiness
}

// Function generates the secret name based on the cluster name
func (rs *RemoteSecretManager) getFullSecretName(cd *kcmv1beta1.ClusterDeployment) string {
	if utils.IsAdopted(cd) {
		return fmt.Sprintf("%s-kubeconf", cd.Name)
	}
	return fmt.Sprintf("%s-kubeconfig", cd.Name)
}

func (rs *RemoteSecretManager) remoteSecretExists(ctx context.Context, cd *kcmv1beta1.ClusterDeployment) (bool, error) {
	secret := &corev1.Secret{}
	secretName := GetRemoteSecretName(cd.Name, cd.Namespace)
	return utils.IsResourceExists(ctx, rs.client, secret, secretName, istio.IstioSystemNamespace)
}

// Function creates the remote secret resource in k8s
func (rs *RemoteSecretManager) createSecretResource(ctx context.Context, secret *corev1.Secret) error {
	if err := rs.client.Create(ctx, secret); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

func (rs *RemoteSecretManager) sendCreationEvent(cd *kcmv1beta1.ClusterDeployment) {
	record.Eventf(
		cd,
		utils.GetEventsAnnotations(cd),
		"SecretCreated",
		"Istio remote secret '%s' is successfully created",
		GetRemoteSecretName(cd.Name, cd.Namespace),
	)
}

func (rs *RemoteSecretManager) sendDeletionEvent(req ctrl.Request) {
	cd := utils.GetClusterDeploymentStub(req.Name, req.Namespace)
	record.Eventf(
		cd,
		nil,
		"SecretDeleted",
		"Istio remote secret '%s' is successfully deleted",
		GetRemoteSecretName(cd.Name, cd.Namespace),
	)
}

type IstioRemoteSecretCreator struct{}

type IIstioRemoteSecretCreator interface {
	GetRemoteSecret(context.Context, []byte, *kcmv1beta1.ClusterDeployment) (*corev1.Secret, error)
}

func NewIstioRemoteSecret() IIstioRemoteSecretCreator {
	return &IstioRemoteSecretCreator{}
}

// Function creates a remote secret for Istio using the provided kubeconfig
func (rs *IstioRemoteSecretCreator) GetRemoteSecret(ctx context.Context, kubeconfig []byte, clusterDeployment *kcmv1beta1.ClusterDeployment) (*corev1.Secret, error) {
	log := log.FromContext(ctx)

	config, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		log.Error(err, "failed to create new client config")
		return nil, err
	}

	kubeClient, err := kube.NewCLIClient(config)
	if err != nil {
		log.Error(err, "failed to create cli client")
		return nil, err
	}

	secret, warn, err := CreateRemoteSecret(
		ctx,
		multicluster.RemoteSecretOptions{
			Type:                 multicluster.SecretTypeRemote,
			AuthType:             multicluster.RemoteSecretAuthTypeBearerToken,
			ClusterName:          clusterDeployment.Name,
			CreateServiceAccount: false,
			KubeOptions: multicluster.KubeOptions{
				Namespace: istio.IstioSystemNamespace,
			},
		},
		clusterDeployment.Namespace,
		kubeClient)
	if err != nil {
		log.Error(err, "failed to create remote secret")
		return nil, err
	}

	if warn != nil {
		log.Info("warning when generating remote secret", "warning", warn)
	}

	return secret, nil
}

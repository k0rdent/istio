package multicluster

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/istio/istio-operator/internal/controller/istio"
	remotesecret "github.com/k0rdent/istio/istio-operator/internal/controller/istio/remote-secret"
	"github.com/k0rdent/istio/istio-operator/internal/controller/record"
	"github.com/k0rdent/istio/istio-operator/internal/controller/utils"
	addoncontrollerv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RemoteSecretPropagationManager struct {
	client client.Client
}

func New(c client.Client) *RemoteSecretPropagationManager {
	return &RemoteSecretPropagationManager{
		client: c,
	}
}

func (m *RemoteSecretPropagationManager) TryCreate(ctx context.Context, clusterDeployment *kcmv1beta1.ClusterDeployment) error {
	log := log.FromContext(ctx)

	exists, err := m.multiClusterServiceExists(ctx, clusterDeployment)
	if err != nil {
		return fmt.Errorf("failed to check MultiClusterService existence: %v", err)
	}

	if exists {
		log.Info("MultiClusterService already exists")
		return nil
	}

	log.Info("Trying to create MultiClusterService for secret propagation")
	if err := m.createMultiClusterService(ctx, clusterDeployment); err != nil {
		return fmt.Errorf("failed to create MultiClusterService resource: %v", err)
	}

	m.sendCreationEvent(clusterDeployment)
	log.Info("MultiClusterService successfully created")
	return nil
}

func (m *RemoteSecretPropagationManager) TryDelete(ctx context.Context, req ctrl.Request) error {
	log := log.FromContext(ctx)

	mcs := &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name: GetMultiClusterServiceName(req.Name, req.Namespace),
		},
	}

	log.Info("Trying to delete MultiClusterService for secret propagation")
	if err := m.client.Delete(ctx, mcs); err != nil {
		if errors.IsNotFound(err) {
			log.Info("MultiClusterService already deleted")
			return nil
		}
		return fmt.Errorf("failed to delete MultiClusterService: %v", err)
	}

	m.sendDeletionEvent(req)
	log.Info("MultiClusterService successfully deleted")
	return nil
}

func (m *RemoteSecretPropagationManager) multiClusterServiceExists(ctx context.Context, cd *kcmv1beta1.ClusterDeployment) (bool, error) {
	mcs := &kcmv1beta1.MultiClusterService{}
	mcsName := GetMultiClusterServiceName(cd.Name, cd.Namespace)
	return utils.IsResourceExists(ctx, m.client, mcs, mcsName, "")
}

func (m *RemoteSecretPropagationManager) createMultiClusterService(ctx context.Context, cd *kcmv1beta1.ClusterDeployment) error {
	mcs := &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name: GetMultiClusterServiceName(cd.Name, cd.Namespace),
			Labels: map[string]string{
				utils.ManagedByLabel: utils.ManagedByValue,
				"cluster-name":       cd.Name,
				"cluster-namespace":  cd.Namespace,
			},
		},
		Spec: kcmv1beta1.MultiClusterServiceSpec{
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					istio.IstioRoleLabel: "member",
				},
			},
			ServiceSpec: kcmv1beta1.ServiceSpec{
				Services: []kcmv1beta1.Service{
					{
						Name:      "istio-secret-propagation",
						Namespace: istio.IstioSystemNamespace,
						Template:  fmt.Sprintf("%s-base-propagation", istio.IstioReleaseName),
					},
				},
				TemplateResourceRefs: []addoncontrollerv1beta1.TemplateResourceRef{
					{
						Identifier: "Secret",
						Resource: corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       remotesecret.GetRemoteSecretName(cd.Name, cd.Namespace),
							Namespace:  istio.IstioSystemNamespace,
						},
					},
				},
			},
		},
	}

	if err := m.client.Create(ctx, mcs); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

func (m *RemoteSecretPropagationManager) sendCreationEvent(cd *kcmv1beta1.ClusterDeployment) {
	record.Eventf(
		cd,
		utils.GetEventsAnnotations(cd),
		"MultiClusterServiceCreated",
		"MultiClusterService '%s' for secret propagation is successfully created",
		GetMultiClusterServiceName(cd.Name, cd.Namespace),
	)
}

func (m *RemoteSecretPropagationManager) sendDeletionEvent(req ctrl.Request) {
	cd := utils.GetClusterDeploymentStub(req.Name, req.Namespace)
	record.Eventf(
		cd,
		nil,
		"MultiClusterServiceDeleted",
		"MultiClusterService '%s' for secret propagation is successfully deleted",
		GetMultiClusterServiceName(req.Name, req.Namespace),
	)
}

func GetMultiClusterServiceName(clusterName, namespace string) string {
	name := fmt.Sprintf("%s-%s", namespace, clusterName)
	return utils.GetNameHash("remote-secret-propagation", name)
}

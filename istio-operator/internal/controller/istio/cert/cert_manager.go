package cert

import (
	"context"

	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/k0rdent/istio/istio-operator/internal/controller/istio"
	"github.com/k0rdent/istio/istio-operator/internal/controller/record"
	"github.com/k0rdent/istio/istio-operator/internal/controller/utils"
	"github.com/k0rdent/istio/istio-operator/internal/k8s"
	addoncontrollerv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type CertManager struct {
	k8sClient client.Client
}

func New(client client.Client) *CertManager {
	return &CertManager{
		k8sClient: client,
	}
}

func (cm *CertManager) TryCreate(ctx context.Context, clusterDeployment *kcmv1beta1.ClusterDeployment) error {
	log := log.FromContext(ctx)
	log.Info("Trying to create certificate")

	cert := cm.generateClusterCACertificate(clusterDeployment)
	if err := cm.createCertificate(ctx, cert, clusterDeployment); err != nil {
		return fmt.Errorf("failed to create istio certificate: %v", err)
	}

	if !utils.IsClusterDeploymentReady(clusterDeployment) {
		log.Info("Cluster deployment is not ready")
		return nil
	}

	if !utils.IsInMesh(clusterDeployment) {
		return nil
	}

	createdInKCMRegion, err := k8s.CreatedInKCMRegion(ctx, cm.k8sClient, clusterDeployment)
	if err != nil {
		return fmt.Errorf("failed to determine cluster region: %v", err)
	}

	if !createdInKCMRegion {
		return nil
	}

	log.Info("Trying to create MultiClusterService for certificate propagation to region clusters")
	if err := cm.createCaMultiClusterService(ctx, clusterDeployment); err != nil {
		return fmt.Errorf("failed to create MultiClusterService for certificate propagation: %v", err)
	}

	return nil
}

func (cm *CertManager) TryDelete(ctx context.Context, req ctrl.Request) error {
	certName := GetCertName(req.Name, req.Namespace)
	log := log.FromContext(ctx)

	log.Info("Trying to delete istio certificate", "certificateName", certName)
	if err := cm.k8sClient.Delete(ctx, &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName,
			Namespace: istio.IstioSystemNamespace,
		},
	}); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Istio Certificate already deleted", "certificateName", certName)
			return nil
		}
		return fmt.Errorf("failed to delete istio certificate")
	}

	log.Info("Trying to delete MultiClusterService for certificate propagation")
	if err := cm.k8sClient.Delete(ctx, &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name: GetCertNameHash(req.Name, req.Namespace),
		},
	}); err != nil {
		if errors.IsNotFound(err) {
			log.Info("MultiClusterService for certificate propagation already deleted")
			return nil
		}
		return fmt.Errorf("failed to delete MultiClusterService: %v", err)
	}

	log.Info("Istio Certificate successfully deleted", "certificateName", certName)
	cm.sendDeletionEvent(req)
	return nil
}

func (cm *CertManager) createCertificate(ctx context.Context, cert *cmv1.Certificate, clusterDeployment *kcmv1beta1.ClusterDeployment) error {
	log := log.FromContext(ctx)
	log.Info("Creating Intermediate Istio CA certificate", "certificateName", cert.Name)

	if err := cm.k8sClient.Create(ctx, cert); err != nil {
		if errors.IsAlreadyExists(err) {
			log.Info("Istio CA certificate already exists", "certificateName", cert.Name)
			return nil
		}
		return err
	}
	cm.sendCreationEvent(clusterDeployment)
	return nil
}

func (cm *CertManager) generateClusterCACertificate(cd *kcmv1beta1.ClusterDeployment) *cmv1.Certificate {
	certName := GetCertName(cd.Name, cd.Namespace)

	return &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName,
			Namespace: istio.IstioSystemNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "istio-operator",
			},
		},
		Spec: cmv1.CertificateSpec{
			IsCA:       true,
			CommonName: fmt.Sprintf("%s CA", cd.Name),
			Subject: &cmv1.X509Subject{
				Organizations: []string{"Istio"},
			},
			PrivateKey: &cmv1.CertificatePrivateKey{
				Algorithm: cmv1.ECDSAKeyAlgorithm,
				Size:      521,
			},
			SecretName: certName,
			IssuerRef: cmmetav1.ObjectReference{
				Name:  fmt.Sprintf("%s-root", istio.IstioReleaseName),
				Kind:  "Issuer",
				Group: "cert-manager.io",
			},
		},
	}
}

// This function creates MultiClusterService to propagate CA certificate to KCM Region clusters with specific Mesh
// TODO: Remove this function once KCM implements automatic copying of the required resources to region clusters.
func (cm *CertManager) createCaMultiClusterService(ctx context.Context, cd *kcmv1beta1.ClusterDeployment) error {
	mcs := &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name: GetCertNameHash(cd.Name, cd.Namespace),
			Labels: map[string]string{
				utils.ManagedByLabel: utils.ManagedByValue,
				"cluster-name":       cd.Name,
				"cluster-namespace":  cd.Namespace,
			},
		},
		Spec: kcmv1beta1.MultiClusterServiceSpec{
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					utils.IstioMeshLabel:                      cd.Labels[utils.IstioMeshLabel],
					"k0rdent.mirantis.com/kcm-region-cluster": "true",
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
						Identifier: "Data",
						Resource: corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       GetCertName(cd.Name, cd.Namespace),
							Namespace:  istio.IstioSystemNamespace,
						},
					},
				},
			},
		},
	}

	if err := cm.k8sClient.Create(ctx, mcs); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

func (cm *CertManager) sendCreationEvent(cd *kcmv1beta1.ClusterDeployment) {
	record.Eventf(
		cd,
		utils.GetEventsAnnotations(cd),
		"CertificateCreated",
		"Istio certificate '%s' is successfully created",
		GetCertName(cd.Name, cd.Namespace),
	)
}

func (cm *CertManager) sendDeletionEvent(req ctrl.Request) {
	cd := utils.GetClusterDeploymentStub(req.Name, req.Namespace)
	record.Eventf(
		cd,
		nil,
		"CertificateDeleted",
		"Istio certificate '%s' is successfully deleted",
		GetCertName(cd.Name, cd.Namespace),
	)
}

func GetCertName(clusterName, namespace string) string {
	return fmt.Sprintf("%s-%s-%s-ca", istio.IstioReleaseName, namespace, clusterName)
}

func GetCertNameHash(clusterName, namespace string) string {
	return utils.GetNameHash("ca-cert-propagation", GetCertName(clusterName, namespace))
}

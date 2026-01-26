package secretrotation

import (
	"context"
	"fmt"
	"time"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/istio/istio-operator/internal/controller/istio"
	remotesecret "github.com/k0rdent/istio/istio-operator/internal/controller/istio/remote-secret"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// RotationInterval defines how often secrets should be rotated
	RotationInterval = 1 * time.Hour
)

// Manager handles periodic rotation of Istio remote secrets.
// It runs a background worker that periodically updates remote secrets for all
// clusters labeled with the Istio role label, ensuring service account tokens
// remain fresh and preventing expiration-related connectivity issues.
type Manager struct {
	kubeClient          client.Client
	ticker              *time.Ticker
	remoteSecretManager *remotesecret.RemoteSecretManager
}

// NewManager creates a new secret rotation manager
func NewManager(kubeClient client.Client) *Manager {
	return &Manager{
		kubeClient:          kubeClient,
		ticker:              time.NewTicker(RotationInterval),
		remoteSecretManager: remotesecret.New(kubeClient),
	}
}

// StartRotationWorker starts a background worker that periodically rotates Istio remote secrets.
// The worker runs on a timer defined by RotationInterval and updates secrets for all clusters
// with the Istio role label. It continues running until the provided context is cancelled.
// This method blocks until the context is done, so it should typically be run in a goroutine.
func (m *Manager) StartRotationWorker(ctx context.Context) {
	log := log.FromContext(ctx)
	log.Info("Starting secret rotation worker", "interval", RotationInterval)

	defer m.ticker.Stop()

	for {
		select {
		case <-m.ticker.C:
			m.rotateSecrets(ctx)
		case <-ctx.Done():
			log.Info("Secret rotation worker stopped")
			return
		}
	}
}

func (m *Manager) rotateSecrets(ctx context.Context) {
	log := log.FromContext(ctx)
	log.Info("Starting secret rotation cycle")

	clusters, err := m.getIstioClusters(ctx)
	if err != nil {
		log.Error(err, "Failed to get Istio clusters for secret rotation")
		return
	}

	if len(clusters) == 0 {
		log.Info("No Istio clusters found for secret rotation")
		return
	}

	log.Info("Rotating secrets for clusters", "count", len(clusters))

	// Rotate secrets for each cluster using AllowOverwrite option
	opt := remotesecret.CreateOptions{AllowOverwrite: true}
	successCount := 0
	for _, cluster := range clusters {
		if err := m.remoteSecretManager.TryCreate(ctx, &cluster, opt); err != nil {
			log.Error(err, "Failed to rotate secret for cluster", "cluster", cluster.Name, "namespace", cluster.Namespace)
		} else {
			successCount++
		}
	}

	log.Info("Secret rotation cycle completed", "total", len(clusters), "successful", successCount, "failed", len(clusters)-successCount)
}

func (m *Manager) getIstioClusters(ctx context.Context) ([]kcmv1beta1.ClusterDeployment, error) {
	clustersList := kcmv1beta1.ClusterDeploymentList{}

	requirement, err := labels.NewRequirement(istio.IstioRoleLabel, selection.Exists, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create label requirement: %w", err)
	}
	selector := labels.NewSelector().Add(*requirement)
	opts := &client.ListOptions{LabelSelector: selector}

	if err := m.kubeClient.List(ctx, &clustersList, opts); err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	return clustersList.Items, nil
}

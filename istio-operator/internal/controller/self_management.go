/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/istio/istio-operator/internal/controller/istio/cert"
	"github.com/k0rdent/istio/istio-operator/internal/controller/istio/multicluster"
	remotesecret "github.com/k0rdent/istio/istio-operator/internal/controller/istio/remote-secret"
	"github.com/k0rdent/istio/istio-operator/internal/k8s"
	secretrotation "github.com/k0rdent/istio/istio-operator/internal/secret-rotation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const selfManagementRetryDelay = 15 * time.Second

// SelfManagementReconciler creates Istio resources (remote secret, CA certificate, and propagation MCS)
// for the management cluster itself.
// It is registered as a manager Runnable and executes once at startup, retrying on failure.
type SelfManagementReconciler struct {
	LocalKubeClient                *k8s.KubeClient
	RemoteSecretManager            *remotesecret.RemoteSecretManager
	IstioCertManager               *cert.CertManager
	RemoteSecretPropagationManager *multicluster.RemoteSecretPropagationManager
	ManagementClusterName          string
	ManagementClusterNamespace     string
	ManagementClusterAPIServer     string
}

// Start creates all management cluster resources on startup (retrying until successful),
// then periodically refreshes the remote secret token.
func (r *SelfManagementReconciler) Start(ctx context.Context) error {
	log := log.FromContext(ctx)
	log.Info("Starting self-management reconciliation",
		"clusterName", r.ManagementClusterName,
		"namespace", r.ManagementClusterNamespace,
	)

	// Initial reconciliation with retry until success.
	for {
		if err := r.reconcile(ctx); err != nil {
			log.Error(err, "Self-management reconciliation failed, retrying",
				"retryDelay", selfManagementRetryDelay,
			)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(selfManagementRetryDelay):
			}
			continue
		}
		break
	}

	log.Info("Self-management initial reconciliation complete, scheduling periodic token refresh",
		"interval", secretrotation.RotationInterval,
	)

	// Periodic refresh: the remote secret embeds a short-lived service account token
	// so it must be rotated at the same cadence as the other cluster secrets.
	ticker := time.NewTicker(secretrotation.RotationInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := r.reconcile(ctx); err != nil {
				log.Error(err, "Self-management periodic refresh failed")
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (r *SelfManagementReconciler) reconcile(ctx context.Context) error {
	cd := &kcmv1beta1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ManagementClusterName,
			Namespace: r.ManagementClusterNamespace,
		},
	}

	if err := r.RemoteSecretManager.TryCreateForLocalCluster(
		ctx,
		r.LocalKubeClient,
		r.ManagementClusterName,
		r.ManagementClusterNamespace,
		r.ManagementClusterAPIServer,
	); err != nil {
		return err
	}

	if err := r.IstioCertManager.TryCreateForCluster(
		ctx,
		r.ManagementClusterName,
		r.ManagementClusterNamespace,
	); err != nil {
		return err
	}

	if err := r.RemoteSecretPropagationManager.TryCreate(ctx, cd); err != nil {
		return err
	}

	return nil
}

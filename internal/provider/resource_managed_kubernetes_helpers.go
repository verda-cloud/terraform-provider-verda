// Copyright 2026 Verda Cloud Oy
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

func managedKubernetesFormatClusterID(location, name string) string {
	return location + "/" + name
}

func managedKubernetesParseClusterID(id string) (location, name string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected cluster ID in the form <location>/<cluster-name>, got %q", id)
	}
	return parts[0], parts[1], nil
}

func managedKubernetesFormatNodePoolID(location, cluster, pool string) string {
	return location + "/" + cluster + "/" + pool
}

func managedKubernetesParseNodePoolID(id string) (location, cluster, pool string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("expected node pool ID in the form <location>/<cluster-name>/<pool-name>, got %q", id)
	}
	return parts[0], parts[1], parts[2], nil
}

func managedKubernetesParseKubeconfigImportID(id string) (location, cluster, kubeconfigID string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("expected kubeconfig import ID in the form <location>/<cluster-name>/<kubeconfig-id>, got %q", id)
	}
	return parts[0], parts[1], parts[2], nil
}

func managedKubernetesIsNotFound(err error) bool {
	var apiErr *verda.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404
	}
	return false
}

// managedKubernetesPollInterval is how often the waiters below ask the API;
// unit tests shorten it.
var managedKubernetesPollInterval = 10 * time.Second

type managedKubernetesClusterGetter interface {
	Get(ctx context.Context, location, clusterName string) (*verda.ManagedKubernetesCluster, error)
}

type managedKubernetesNodePoolGetter interface {
	GetNodePool(ctx context.Context, location, clusterName, poolName string) (*verda.ManagedKubernetesNodePool, error)
}

type managedKubernetesKubeconfigGetter interface {
	GetKubeconfig(ctx context.Context, location, clusterName, id string) (*verda.ManagedKubernetesKubeconfig, error)
}

// managedKubernetesWaitForClusterRunning polls the cluster until it reports
// phase running, or ctx runs out. "degraded" is not treated as terminal; the
// API does not document it as one.
func managedKubernetesWaitForClusterRunning(ctx context.Context, clusters managedKubernetesClusterGetter, location, name string) (*verda.ManagedKubernetesCluster, error) {
	var last string
	for {
		cluster, err := clusters.Get(ctx, location, name)
		if err == nil {
			last = cluster.Phase
			if cluster.Phase == verda.ManagedKubernetesClusterPhaseRunning {
				return cluster, nil
			}
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("cluster %s/%s not running before the timeout (last phase %q)", location, name, last)
			}
			return nil, ctx.Err()
		case <-time.After(managedKubernetesPollInterval):
		}
	}
}

func managedKubernetesWaitForClusterDeleted(ctx context.Context, clusters managedKubernetesClusterGetter, location, name string) error {
	for {
		_, err := clusters.Get(ctx, location, name)
		if managedKubernetesIsNotFound(err) {
			return nil
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("cluster %s/%s still exists after the delete timeout", location, name)
			}
			return ctx.Err()
		case <-time.After(managedKubernetesPollInterval):
		}
	}
}

func managedKubernetesWaitForNodePoolActive(ctx context.Context, pools managedKubernetesNodePoolGetter, location, cluster, pool string) (*verda.ManagedKubernetesNodePool, error) {
	var last *verda.ManagedKubernetesNodePool
	for {
		p, err := pools.GetNodePool(ctx, location, cluster, pool)
		if err == nil {
			last = p
			if p.State == verda.ManagedKubernetesNodePoolStateActive {
				return p, nil
			}
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("node pool %s/%s/%s not active before the timeout%s",
					location, cluster, pool, managedKubernetesNodePoolDetail(last))
			}
			return nil, ctx.Err()
		case <-time.After(managedKubernetesPollInterval):
		}
	}
}

func managedKubernetesWaitForNodePoolDeleted(ctx context.Context, pools managedKubernetesNodePoolGetter, location, cluster, pool string) error {
	for {
		_, err := pools.GetNodePool(ctx, location, cluster, pool)
		if managedKubernetesIsNotFound(err) {
			return nil
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("node pool %s/%s/%s still exists after the delete timeout", location, cluster, pool)
			}
			return ctx.Err()
		case <-time.After(managedKubernetesPollInterval):
		}
	}
}

func managedKubernetesNodePoolDetail(pool *verda.ManagedKubernetesNodePool) string {
	if pool == nil {
		return ""
	}
	detail := fmt.Sprintf(" (last state %q", pool.State)
	if len(pool.Errors) > 0 {
		parts := make([]string, 0, len(pool.Errors))
		for _, e := range pool.Errors {
			parts = append(parts, e.Reason+": "+e.Message)
		}
		detail += ", errors: " + strings.Join(parts, "; ")
	}
	return detail + ")"
}

func managedKubernetesWaitForKubeconfigReady(ctx context.Context, kubeconfigs managedKubernetesKubeconfigGetter, location, cluster, id string) (*verda.ManagedKubernetesKubeconfig, error) {
	var last string
	for {
		kc, err := kubeconfigs.GetKubeconfig(ctx, location, cluster, id)
		if err == nil {
			last = kc.Status
			if kc.Status == verda.ManagedKubernetesKubeconfigStatusReady {
				return kc, nil
			}
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("kubeconfig %s/%s/%s not ready before the timeout (last status %q)", location, cluster, id, last)
			}
			return nil, ctx.Err()
		case <-time.After(managedKubernetesPollInterval):
		}
	}
}

func managedKubernetesWaitForKubeconfigDeleted(ctx context.Context, kubeconfigs managedKubernetesKubeconfigGetter, location, cluster, id string) error {
	for {
		_, err := kubeconfigs.GetKubeconfig(ctx, location, cluster, id)
		if managedKubernetesIsNotFound(err) {
			return nil
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("kubeconfig %s/%s/%s still exists after the delete timeout", location, cluster, id)
			}
			return ctx.Err()
		case <-time.After(managedKubernetesPollInterval):
		}
	}
}

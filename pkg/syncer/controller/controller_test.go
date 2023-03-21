package controller

import (
	"context"
	"time"

	edgev1alpha1 "github.com/kcp-dev/edge-mc/pkg/syncer/apis/edge/v1alpha1"
	edgefakeclient "github.com/kcp-dev/edge-mc/pkg/syncer/client/clientset/versioned/fake"
	edgeinformers "github.com/kcp-dev/edge-mc/pkg/syncer/client/informers/externalversions"
	"github.com/kcp-dev/logicalcluster/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"testing"
)

var scheme *runtime.Scheme

func init() {
	scheme = runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
}

func TestSyncConfig(t *testing.T) {
	tests := map[string]struct {
		syncConfig *edgev1alpha1.EdgeSyncConfig
	}{
		"Syncer updates downsyncer/upsyncer following to syncConfig": {
			syncConfig: syncConfig("test-sync-config", types.UID("uid")),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			logger := klog.FromContext(ctx)

			syncConfigClientSet := edgefakeclient.NewSimpleClientset(tc.syncConfig)
			syncConfigClient := syncConfigClientSet.EdgeV1alpha1().EdgeSyncConfigs()
			syncConfigInformerFactory := edgeinformers.NewSharedScopedInformerFactoryWithOptions(syncConfigClientSet, 0)
			syncConfigInformer := syncConfigInformerFactory.Edge().V1alpha1().EdgeSyncConfigs()

			controller, err := NewSyncConfigController(logger, syncConfigClient, syncConfigInformer, tc.syncConfig.UID, tc.syncConfig.Name, nil, nil)
			require.NoError(t, err)

			syncConfigInformerFactory.Start(ctx.Done())

			err = nil
			require.Eventually(t, func() bool {
				syncConfig, _err := syncConfigInformer.Lister().Get(tc.syncConfig.Name)
				if _err != nil {
					if errors.IsNotFound(_err) {
						return false
					} else {
						err = _err
						return true
					}
				}
				return syncConfig != nil
			}, wait.ForeverTestTimeout, 1*time.Second)
			assert.NoError(t, err)

			err = controller.process(ctx, tc.syncConfig.Name)
			require.NoError(t, err)
		})
	}
}

func syncConfig(name string, uid types.UID) *edgev1alpha1.EdgeSyncConfig {
	return &edgev1alpha1.EdgeSyncConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  uid,
		},
	}
}

func deployment(name, namespace, clusterName string, labels, annotations map[string]string, finalizers []string) *appsv1.Deployment {
	if clusterName != "" {
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[logicalcluster.AnnotationKey] = clusterName
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
			Finalizers:  finalizers,
		},
	}
}

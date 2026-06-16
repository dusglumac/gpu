package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gpuv1beta1 "github.com/kyma-project/gpu/api/v1beta1"
)

const timeSlicingConfigMapName = "gpu-time-slicing-config"

// reconcileTimeSlicing creates or updates the device plugin ConfigMap when
// spec.timeSlicing is set, and deletes it (if present) when it is nil.
func (r *GpuReconciler) reconcileTimeSlicing(ctx context.Context, gpu *gpuv1beta1.Gpu) error {
	if gpu.Spec.TimeSlicing == nil {
		return r.deleteTimeSlicingConfigMap(ctx)
	}
	return r.applyTimeSlicingConfigMap(ctx, gpu.Spec.TimeSlicing.Replicas)
}

func (r *GpuReconciler) applyTimeSlicingConfigMap(ctx context.Context, replicas int32) error {
	data := fmt.Sprintf("version: v1\nsharing:\n  timeSlicing:\n    resources:\n    - name: nvidia.com/gpu\n      replicas: %d\n", replicas)

	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: timeSlicingConfigMapName, Namespace: gpuOperatorNamespace}, existing)

	if apierrors.IsNotFound(err) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      timeSlicingConfigMapName,
				Namespace: gpuOperatorNamespace,
			},
			Data: map[string]string{"any": data},
		}
		if err := r.Create(ctx, cm); err != nil {
			return fmt.Errorf("creating time-slicing ConfigMap: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("getting time-slicing ConfigMap: %w", err)
	}

	patch := client.MergeFrom(existing.DeepCopy())
	existing.Data = map[string]string{"any": data}
	if err := r.Patch(ctx, existing, patch); err != nil {
		return fmt.Errorf("updating time-slicing ConfigMap: %w", err)
	}
	return nil
}

func (r *GpuReconciler) deleteTimeSlicingConfigMap(ctx context.Context) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      timeSlicingConfigMapName,
			Namespace: gpuOperatorNamespace,
		},
	}
	if err := r.Delete(ctx, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting time-slicing ConfigMap: %w", err)
	}
	return nil
}

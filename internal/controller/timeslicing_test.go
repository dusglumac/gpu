package controller

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	gpuv1beta1 "github.com/kyma-project/gpu/api/v1beta1"
)

func fakeClientWithScheme(objects ...client.Object) client.Client {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = gpuv1beta1.AddToScheme(s)
	return fake.NewClientBuilder().WithScheme(s).WithObjects(objects...).Build()
}

func timeSlicingConfigMap(replicas int, namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      timeSlicingConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"any": fmt.Sprintf("version: v1\nsharing:\n  timeSlicing:\n    resources:\n    - name: nvidia.com/gpu\n      replicas: %d\n", replicas),
		},
	}
}

func TestReconcileTimeSlicing_CreateOnEnable(t *testing.T) {
	c := fakeClientWithScheme()
	r := &GpuReconciler{Client: c}

	gpu := &gpuv1beta1.Gpu{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu"},
		Spec:       gpuv1beta1.GpuSpec{TimeSlicing: &gpuv1beta1.TimeSlicingSpec{Replicas: 4}},
	}

	if err := r.reconcileTimeSlicing(context.Background(), gpu); err != nil {
		t.Fatalf("reconcileTimeSlicing() unexpected error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: timeSlicingConfigMapName, Namespace: gpuOperatorNamespace}, cm); err != nil {
		t.Fatalf("ConfigMap not found after enable: %v", err)
	}
	wantData := "version: v1\nsharing:\n  timeSlicing:\n    resources:\n    - name: nvidia.com/gpu\n      replicas: 4\n"
	if got := cm.Data["any"]; got != wantData {
		t.Errorf("ConfigMap data[any] = %q, want %q", got, wantData)
	}
}

func TestReconcileTimeSlicing_UpdateOnReplicaChange(t *testing.T) {
	existing := timeSlicingConfigMap(4, gpuOperatorNamespace)
	c := fakeClientWithScheme(existing)
	r := &GpuReconciler{Client: c}

	gpu := &gpuv1beta1.Gpu{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu"},
		Spec:       gpuv1beta1.GpuSpec{TimeSlicing: &gpuv1beta1.TimeSlicingSpec{Replicas: 8}},
	}

	if err := r.reconcileTimeSlicing(context.Background(), gpu); err != nil {
		t.Fatalf("reconcileTimeSlicing() unexpected error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: timeSlicingConfigMapName, Namespace: gpuOperatorNamespace}, cm); err != nil {
		t.Fatalf("ConfigMap not found after update: %v", err)
	}
	wantData := "version: v1\nsharing:\n  timeSlicing:\n    resources:\n    - name: nvidia.com/gpu\n      replicas: 8\n"
	if got := cm.Data["any"]; got != wantData {
		t.Errorf("ConfigMap data[any] = %q, want %q", got, wantData)
	}
}

func TestReconcileTimeSlicing_DeleteOnDisable(t *testing.T) {
	existing := timeSlicingConfigMap(4, gpuOperatorNamespace)
	c := fakeClientWithScheme(existing)
	r := &GpuReconciler{Client: c}

	gpu := &gpuv1beta1.Gpu{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu"},
		Spec:       gpuv1beta1.GpuSpec{TimeSlicing: nil},
	}

	if err := r.reconcileTimeSlicing(context.Background(), gpu); err != nil {
		t.Fatalf("reconcileTimeSlicing() unexpected error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	err := c.Get(context.Background(), types.NamespacedName{Name: timeSlicingConfigMapName, Namespace: gpuOperatorNamespace}, cm)
	if !apierrors.IsNotFound(err) {
		t.Errorf("ConfigMap should be deleted, got err=%v", err)
	}
}

func TestReconcileTimeSlicing_NoOpDeleteWhenAbsent(t *testing.T) {
	c := fakeClientWithScheme()
	r := &GpuReconciler{Client: c}

	gpu := &gpuv1beta1.Gpu{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu"},
		Spec:       gpuv1beta1.GpuSpec{TimeSlicing: nil},
	}

	if err := r.reconcileTimeSlicing(context.Background(), gpu); err != nil {
		t.Fatalf("reconcileTimeSlicing() with nil TimeSlicing and absent ConfigMap should not error: %v", err)
	}
}

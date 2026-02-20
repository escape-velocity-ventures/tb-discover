package commands

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func int32Ptr(i int32) *int32 { return &i }

func TestDeletePod(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "target-pod", Namespace: "default"}},
	)
	exec := NewExecutor(clientset)

	result := exec.Execute(context.Background(), Command{
		ID: "cmd-1", Action: "delete_pod",
		TargetKind: "Pod", TargetNamespace: "default", TargetName: "target-pod",
	})

	if !result.Success {
		t.Errorf("expected success: %s", result.Message)
	}

	_, err := clientset.CoreV1().Pods("default").Get(context.Background(), "target-pod", metav1.GetOptions{})
	if err == nil {
		t.Error("pod should have been deleted")
	}
}

func TestForceDeletePod(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "stuck-pod", Namespace: "default"}},
	)
	exec := NewExecutor(clientset)

	result := exec.Execute(context.Background(), Command{
		ID: "cmd-2", Action: "force_delete_pod",
		TargetKind: "Pod", TargetNamespace: "default", TargetName: "stuck-pod",
	})

	if !result.Success {
		t.Errorf("expected success: %s", result.Message)
	}
}

func TestDeleteDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "target-deploy", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "app:latest"}}},
				},
			},
		},
	)
	exec := NewExecutor(clientset)

	result := exec.Execute(context.Background(), Command{
		ID: "cmd-3", Action: "delete_deployment",
		TargetKind: "Deployment", TargetNamespace: "default", TargetName: "target-deploy",
	})

	if !result.Success {
		t.Errorf("expected success: %s", result.Message)
	}
}

func TestDeletePVC(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data-pvc", Namespace: "default"}},
	)
	exec := NewExecutor(clientset)

	result := exec.Execute(context.Background(), Command{
		ID: "cmd-4", Action: "delete_pvc",
		TargetKind: "PersistentVolumeClaim", TargetNamespace: "default", TargetName: "data-pvc",
	})

	if !result.Success {
		t.Errorf("expected success: %s", result.Message)
	}
}

func TestRestartDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(2),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "web"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Image: "web:latest"}}},
				},
			},
		},
	)
	exec := NewExecutor(clientset)

	result := exec.Execute(context.Background(), Command{
		ID: "cmd-5", Action: "restart_deployment",
		TargetKind: "Deployment", TargetNamespace: "default", TargetName: "web",
	})

	if !result.Success {
		t.Errorf("expected success: %s", result.Message)
	}
}

func TestCordonNode(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "worker-1"}},
	)
	exec := NewExecutor(clientset)

	result := exec.Execute(context.Background(), Command{
		ID: "cmd-6", Action: "cordon_node",
		TargetKind: "Node", TargetName: "worker-1",
	})

	if !result.Success {
		t.Errorf("expected success: %s", result.Message)
	}
}

func TestUncordonNode(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
			Spec:       corev1.NodeSpec{Unschedulable: true},
		},
	)
	exec := NewExecutor(clientset)

	result := exec.Execute(context.Background(), Command{
		ID: "cmd-7", Action: "uncordon_node",
		TargetKind: "Node", TargetName: "worker-1",
	})

	if !result.Success {
		t.Errorf("expected success: %s", result.Message)
	}
}

func TestUnknownAction(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	exec := NewExecutor(clientset)

	result := exec.Execute(context.Background(), Command{
		ID: "cmd-99", Action: "fly_to_moon",
		TargetKind: "Pod", TargetNamespace: "default", TargetName: "pod",
	})

	if result.Success {
		t.Error("unknown action should fail")
	}
	if result.Message != "unknown action: fly_to_moon" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestScaleDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(2),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "web"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Image: "web:latest"}}},
				},
			},
		},
	)

	// Register a reactor to handle the /scale subresource
	clientset.PrependReactor("get", "deployments/scale", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 2},
		}, nil
	})
	clientset.PrependReactor("update", "deployments/scale", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 5},
		}, nil
	})

	exec := NewExecutor(clientset)

	result := exec.Execute(context.Background(), Command{
		ID: "cmd-8", Action: "scale",
		TargetKind: "Deployment", TargetNamespace: "default", TargetName: "web",
		Parameters: map[string]any{"replicas": float64(5)},
	})

	if !result.Success {
		t.Errorf("expected success: %s", result.Message)
	}
}

func TestScaleMissingReplicas(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	exec := NewExecutor(clientset)

	result := exec.Execute(context.Background(), Command{
		ID: "cmd-9", Action: "scale",
		TargetKind: "Deployment", TargetNamespace: "default", TargetName: "web",
		Parameters: map[string]any{}, // missing replicas
	})

	if result.Success {
		t.Error("should fail with missing replicas parameter")
	}
}

func TestTuneResourceLimits(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "no-limits", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app", Image: "app:latest"},
						},
					},
				},
			},
		},
	)
	exec := NewExecutor(clientset)

	result := exec.Execute(context.Background(), Command{
		ID: "cmd-10", Action: "tune_resource_limits",
		TargetKind: "Deployment", TargetNamespace: "default", TargetName: "no-limits",
		Parameters: map[string]any{"cpu_limit": "500m", "memory_limit": "512Mi"},
	})

	if !result.Success {
		t.Errorf("expected success: %s", result.Message)
	}
}

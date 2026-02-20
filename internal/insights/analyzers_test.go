package insights

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

func int32Ptr(i int32) *int32 { return &i }

func TestStalePodAnalyzer(t *testing.T) {
	now := time.Now()
	twoHoursAgo := metav1.NewTime(now.Add(-2 * time.Hour))
	fiveMinAgo := metav1.NewTime(now.Add(-5 * time.Minute))

	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "old-succeeded", Namespace: "default"},
			Status: corev1.PodStatus{
				Phase:     corev1.PodSucceeded,
				StartTime: &twoHoursAgo,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "recent-succeeded", Namespace: "default"},
			Status: corev1.PodStatus{
				Phase:     corev1.PodSucceeded,
				StartTime: &fiveMinAgo,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "running-pod", Namespace: "default"},
			Status: corev1.PodStatus{
				Phase:     corev1.PodRunning,
				StartTime: &twoHoursAgo,
			},
		},
	)

	a := NewStalePodAnalyzer()
	insights, err := a.Analyze(context.Background(), clientset, "default")
	if err != nil {
		t.Fatal(err)
	}

	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if insights[0].TargetName != "old-succeeded" {
		t.Errorf("expected old-succeeded, got %s", insights[0].TargetName)
	}
	if !insights[0].AutoRemediable {
		t.Error("stale pod should be auto-remediable")
	}
	if insights[0].ProposedAction != "delete_pod" {
		t.Errorf("expected delete_pod action, got %s", insights[0].ProposedAction)
	}
}

func TestStuckTerminatingAnalyzer(t *testing.T) {
	now := time.Now()
	twentyMinAgo := metav1.NewTime(now.Add(-20 * time.Minute))

	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "stuck-pod",
				Namespace:         "default",
				DeletionTimestamp: &twentyMinAgo,
				Finalizers:        []string{"test"},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "normal-pod", Namespace: "default"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
	)

	a := NewStuckTerminatingAnalyzer()
	insights, err := a.Analyze(context.Background(), clientset, "default")
	if err != nil {
		t.Fatal(err)
	}

	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if insights[0].ProposedAction != "force_delete_pod" {
		t.Errorf("expected force_delete_pod, got %s", insights[0].ProposedAction)
	}
}

func TestEvictedPodAnalyzer(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "evicted-pod", Namespace: "default"},
			Status:     corev1.PodStatus{Reason: "Evicted", Phase: corev1.PodFailed},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "normal-pod", Namespace: "default"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
	)

	a := NewEvictedPodAnalyzer()
	insights, err := a.Analyze(context.Background(), clientset, "default")
	if err != nil {
		t.Fatal(err)
	}

	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if insights[0].TargetName != "evicted-pod" {
		t.Errorf("expected evicted-pod, got %s", insights[0].TargetName)
	}
}

func TestMissingProbesAnalyzer(t *testing.T) {
	port := intstr.FromInt32(8080)
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "no-probes", Namespace: "default"},
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
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "has-probes", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test2"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test2"}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: "app:latest",
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{Path: "/healthz", Port: port},
									},
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{Path: "/healthz", Port: port},
									},
								},
							},
						},
					},
				},
			},
		},
	)

	a := NewMissingProbesAnalyzer()
	insights, err := a.Analyze(context.Background(), clientset, "default")
	if err != nil {
		t.Fatal(err)
	}

	if len(insights) != 1 {
		t.Fatalf("expected 1 insight (no-probes), got %d", len(insights))
	}
	if insights[0].TargetName != "no-probes" {
		t.Errorf("expected no-probes, got %s", insights[0].TargetName)
	}
}

func TestUnreadyWorkloadsAnalyzer(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "partial", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(3),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "app:latest"}}},
				},
			},
			Status: appsv1.DeploymentStatus{ReadyReplicas: 1},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "healthy", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(2),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test2"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test2"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "app:latest"}}},
				},
			},
			Status: appsv1.DeploymentStatus{ReadyReplicas: 2},
		},
	)

	a := NewUnreadyWorkloadsAnalyzer()
	insights, err := a.Analyze(context.Background(), clientset, "default")
	if err != nil {
		t.Fatal(err)
	}

	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if insights[0].TargetName != "partial" {
		t.Errorf("expected partial, got %s", insights[0].TargetName)
	}
}

func TestImagePullIssuesAnalyzer(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pull-fail", Namespace: "default"},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  "app",
						Image: "bad-registry.example.com/app:latest",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"},
						},
					},
				},
			},
		},
	)

	a := NewImagePullIssuesAnalyzer()
	insights, err := a.Analyze(context.Background(), clientset, "default")
	if err != nil {
		t.Fatal(err)
	}

	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if insights[0].Severity != "action" {
		t.Errorf("expected action severity, got %s", insights[0].Severity)
	}
}

func TestMissingLimitsAnalyzer(t *testing.T) {
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
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "has-limits", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test2"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test2"}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: "app:latest",
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("256Mi"),
									},
								},
							},
						},
					},
				},
			},
		},
	)

	a := NewMissingLimitsAnalyzer()
	insights, err := a.Analyze(context.Background(), clientset, "default")
	if err != nil {
		t.Fatal(err)
	}

	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if insights[0].TargetName != "no-limits" {
		t.Errorf("expected no-limits, got %s", insights[0].TargetName)
	}
	if insights[0].ProposedAction != "tune_resource_limits" {
		t.Errorf("expected tune_resource_limits, got %s", insights[0].ProposedAction)
	}
}

func TestResourcePressureAnalyzer(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "pressure-node"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue, Message: "memory low"},
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "healthy-node"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
	)

	a := NewResourcePressureAnalyzer()
	insights, err := a.Analyze(context.Background(), clientset, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if insights[0].TargetName != "pressure-node" {
		t.Errorf("expected pressure-node, got %s", insights[0].TargetName)
	}
}

func TestCrashloopingAnalyzer(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "crash-pod-abc-xyz", Namespace: "default"},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "app",
						RestartCount: 50,
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
						},
					},
				},
			},
		},
	)

	a := NewCrashloopingAnalyzer()
	insights, err := a.Analyze(context.Background(), clientset, "default")
	if err != nil {
		t.Fatal(err)
	}

	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if insights[0].Severity != "action" {
		t.Errorf("expected action severity, got %s", insights[0].Severity)
	}
}

// Suppress unused import warnings
var _ = intstr.FromInt32

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// Executor runs commands against a Kubernetes cluster.
type Executor struct {
	clientset kubernetes.Interface
	log       *slog.Logger
}

// NewExecutor creates a new command executor.
func NewExecutor(clientset kubernetes.Interface) *Executor {
	return &Executor{
		clientset: clientset,
		log:       slog.Default().With("component", "command-executor"),
	}
}

// Execute runs a single command and returns the result.
func (e *Executor) Execute(ctx context.Context, cmd Command) CommandResult {
	e.log.Info("executing command",
		"id", cmd.ID, "action", cmd.Action,
		"kind", cmd.TargetKind, "ns", cmd.TargetNamespace, "name", cmd.TargetName)

	var result CommandResult
	switch cmd.Action {
	case "delete_pod":
		result = e.deletePod(ctx, cmd)
	case "force_delete_pod":
		result = e.forceDeletePod(ctx, cmd)
	case "restart_deployment":
		result = e.restartDeployment(ctx, cmd)
	case "scale":
		result = e.scale(ctx, cmd)
	case "delete_deployment":
		result = e.deleteDeployment(ctx, cmd)
	case "delete_pvc":
		result = e.deletePVC(ctx, cmd)
	case "cordon_node":
		result = e.cordonNode(ctx, cmd, true)
	case "uncordon_node":
		result = e.cordonNode(ctx, cmd, false)
	case "tune_resource_limits":
		result = e.tuneResourceLimits(ctx, cmd)
	default:
		result = CommandResult{
			Success: false,
			Message: fmt.Sprintf("unknown action: %s", cmd.Action),
		}
	}

	e.log.Info("command result",
		"id", cmd.ID, "success", result.Success, "message", result.Message)
	return result
}

func (e *Executor) deletePod(ctx context.Context, cmd Command) CommandResult {
	err := e.clientset.CoreV1().Pods(cmd.TargetNamespace).Delete(ctx, cmd.TargetName, metav1.DeleteOptions{})
	if err != nil {
		return CommandResult{Success: false, Message: err.Error()}
	}
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Pod %s/%s deleted", cmd.TargetNamespace, cmd.TargetName),
	}
}

func (e *Executor) forceDeletePod(ctx context.Context, cmd Command) CommandResult {
	grace := int64(0)
	err := e.clientset.CoreV1().Pods(cmd.TargetNamespace).Delete(ctx, cmd.TargetName, metav1.DeleteOptions{
		GracePeriodSeconds: &grace,
	})
	if err != nil {
		return CommandResult{Success: false, Message: err.Error()}
	}
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Pod %s/%s force-deleted (gracePeriod=0)", cmd.TargetNamespace, cmd.TargetName),
	}
}

func (e *Executor) restartDeployment(ctx context.Context, cmd Command) CommandResult {
	restartedAt := time.Now().UTC().Format(time.RFC3339)
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, restartedAt)

	_, err := e.clientset.AppsV1().Deployments(cmd.TargetNamespace).Patch(
		ctx, cmd.TargetName, apitypes.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return CommandResult{Success: false, Message: err.Error()}
	}
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Deployment %s/%s restarted (rollout triggered at %s)", cmd.TargetNamespace, cmd.TargetName, restartedAt),
	}
}

func (e *Executor) scale(ctx context.Context, cmd Command) CommandResult {
	replicasRaw, ok := cmd.Parameters["replicas"]
	if !ok {
		return CommandResult{Success: false, Message: "missing 'replicas' parameter"}
	}
	// JSON numbers decode as float64
	replicasFloat, ok := replicasRaw.(float64)
	if !ok {
		return CommandResult{Success: false, Message: fmt.Sprintf("invalid replicas value: %v", replicasRaw)}
	}
	newReplicas := int32(replicasFloat)
	if newReplicas < 0 {
		return CommandResult{Success: false, Message: fmt.Sprintf("invalid replicas value: %d", newReplicas)}
	}

	// Get current scale
	scale, err := e.clientset.AppsV1().Deployments(cmd.TargetNamespace).GetScale(ctx, cmd.TargetName, metav1.GetOptions{})
	if err != nil {
		return CommandResult{Success: false, Message: err.Error()}
	}
	oldReplicas := scale.Spec.Replicas

	// Update scale
	scale.Spec.Replicas = newReplicas
	_, err = e.clientset.AppsV1().Deployments(cmd.TargetNamespace).UpdateScale(ctx, cmd.TargetName, scale, metav1.UpdateOptions{})
	if err != nil {
		return CommandResult{Success: false, Message: err.Error()}
	}

	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Deployment %s/%s scaled from %d to %d", cmd.TargetNamespace, cmd.TargetName, oldReplicas, newReplicas),
		Details: map[string]any{"old_replicas": oldReplicas, "new_replicas": newReplicas},
	}
}

func (e *Executor) deleteDeployment(ctx context.Context, cmd Command) CommandResult {
	err := e.clientset.AppsV1().Deployments(cmd.TargetNamespace).Delete(ctx, cmd.TargetName, metav1.DeleteOptions{})
	if err != nil {
		return CommandResult{Success: false, Message: err.Error()}
	}
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Deployment %s/%s deleted", cmd.TargetNamespace, cmd.TargetName),
	}
}

func (e *Executor) deletePVC(ctx context.Context, cmd Command) CommandResult {
	err := e.clientset.CoreV1().PersistentVolumeClaims(cmd.TargetNamespace).Delete(ctx, cmd.TargetName, metav1.DeleteOptions{})
	if err != nil {
		return CommandResult{Success: false, Message: err.Error()}
	}
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("PVC %s/%s deleted", cmd.TargetNamespace, cmd.TargetName),
	}
}

func (e *Executor) cordonNode(ctx context.Context, cmd Command, cordon bool) CommandResult {
	patch := fmt.Sprintf(`{"spec":{"unschedulable":%t}}`, cordon)
	_, err := e.clientset.CoreV1().Nodes().Patch(
		ctx, cmd.TargetName, apitypes.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return CommandResult{Success: false, Message: err.Error()}
	}

	action := "cordoned"
	if !cordon {
		action = "uncordoned"
	}
	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("Node %s %s", cmd.TargetName, action),
	}
}

func (e *Executor) tuneResourceLimits(ctx context.Context, cmd Command) CommandResult {
	ns := cmd.TargetNamespace
	name := cmd.TargetName
	kind := cmd.TargetKind

	cpuLimit := "250m"
	memLimit := "256Mi"
	if v, ok := cmd.Parameters["cpu_limit"].(string); ok && v != "" {
		cpuLimit = v
	}
	if v, ok := cmd.Parameters["memory_limit"].(string); ok && v != "" {
		memLimit = v
	}

	// Build a strategic merge patch that sets limits on all containers
	// We need to get the current spec first to know container names
	var containers []map[string]any

	switch kind {
	case "Deployment":
		dep, err := e.clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return CommandResult{Success: false, Message: err.Error()}
		}
		for _, c := range dep.Spec.Template.Spec.Containers {
			memLim := c.Resources.Limits.Memory()
			cpuLim := c.Resources.Limits.Cpu()
			if (memLim != nil && !memLim.IsZero()) && (cpuLim != nil && !cpuLim.IsZero()) {
				continue // already has both limits
			}
			patch := map[string]any{"name": c.Name, "resources": map[string]any{"limits": map[string]string{}}}
			limits := patch["resources"].(map[string]any)["limits"].(map[string]string)
			if memLim == nil || memLim.IsZero() {
				limits["memory"] = memLimit
			}
			if cpuLim == nil || cpuLim.IsZero() {
				limits["cpu"] = cpuLimit
			}
			containers = append(containers, patch)
		}
	case "StatefulSet":
		sts, err := e.clientset.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return CommandResult{Success: false, Message: err.Error()}
		}
		for _, c := range sts.Spec.Template.Spec.Containers {
			memLim := c.Resources.Limits.Memory()
			cpuLim := c.Resources.Limits.Cpu()
			if (memLim != nil && !memLim.IsZero()) && (cpuLim != nil && !cpuLim.IsZero()) {
				continue
			}
			patch := map[string]any{"name": c.Name, "resources": map[string]any{"limits": map[string]string{}}}
			limits := patch["resources"].(map[string]any)["limits"].(map[string]string)
			if memLim == nil || memLim.IsZero() {
				limits["memory"] = memLimit
			}
			if cpuLim == nil || cpuLim.IsZero() {
				limits["cpu"] = cpuLimit
			}
			containers = append(containers, patch)
		}
	case "DaemonSet":
		ds, err := e.clientset.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return CommandResult{Success: false, Message: err.Error()}
		}
		for _, c := range ds.Spec.Template.Spec.Containers {
			memLim := c.Resources.Limits.Memory()
			cpuLim := c.Resources.Limits.Cpu()
			if (memLim != nil && !memLim.IsZero()) && (cpuLim != nil && !cpuLim.IsZero()) {
				continue
			}
			patch := map[string]any{"name": c.Name, "resources": map[string]any{"limits": map[string]string{}}}
			limits := patch["resources"].(map[string]any)["limits"].(map[string]string)
			if memLim == nil || memLim.IsZero() {
				limits["memory"] = memLimit
			}
			if cpuLim == nil || cpuLim.IsZero() {
				limits["cpu"] = cpuLimit
			}
			containers = append(containers, patch)
		}
	default:
		return CommandResult{Success: false, Message: fmt.Sprintf("unsupported kind: %s", kind)}
	}

	if len(containers) == 0 {
		return CommandResult{
			Success: true,
			Message: fmt.Sprintf("%s %s/%s already has all resource limits set", kind, ns, name),
		}
	}

	patchBody := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": containers,
				},
			},
		},
	}
	patchBytes, _ := json.Marshal(patchBody)

	var err error
	switch kind {
	case "Deployment":
		_, err = e.clientset.AppsV1().Deployments(ns).Patch(ctx, name, apitypes.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	case "StatefulSet":
		_, err = e.clientset.AppsV1().StatefulSets(ns).Patch(ctx, name, apitypes.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	case "DaemonSet":
		_, err = e.clientset.AppsV1().DaemonSets(ns).Patch(ctx, name, apitypes.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	}

	if err != nil {
		return CommandResult{Success: false, Message: err.Error()}
	}

	return CommandResult{
		Success: true,
		Message: fmt.Sprintf("%s %s/%s limits set (cpu=%s, memory=%s) for %d container(s)", kind, ns, name, cpuLimit, memLimit, len(containers)),
		Details: map[string]any{"patched_containers": len(containers), "cpu_limit": cpuLimit, "memory_limit": memLimit},
	}
}

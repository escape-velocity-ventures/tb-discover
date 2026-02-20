package scanner

import (
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterScanResultJSONShape(t *testing.T) {
	replicas := int32(3)
	ready := int32(2)
	avail := int32(2)
	desired := int32(7)
	numReady := int32(7)
	minAvail := "1"
	lastSched := "2026-02-19T10:00:00Z"

	result := ClusterScanResult{
		Name:     "test-cluster",
		Provider: "k3s",
		Version:  "v1.34.3+k3s3",
		Nodes: []NodeScanResult{
			{
				Name:    "node-1",
				Status:  "Ready",
				Roles:   []string{"control-plane", "etcd"},
				Version: "v1.34.3+k3s3",
				OS:      "linux",
				OSImage: "Ubuntu 24.04.3 LTS",
			},
		},
		Namespaces: []NamespaceScanResult{
			{
				Name:   "default",
				Labels: map[string]string{"kubernetes.io/metadata.name": "default"},
				Workloads: []WorkloadScanResult{
					{
						Name:              "nginx",
						Namespace:         "default",
						Kind:              "Deployment",
						Replicas:          &replicas,
						ReadyReplicas:     &ready,
						AvailableReplicas: &avail,
						Strategy:          "RollingUpdate",
						Containers: []ContainerInfoK8s{
							{Name: "nginx", Image: "nginx:1.27"},
						},
						Requests: &ResourceRequirements{CPUMillicores: 100, MemoryBytes: 67108864},
						Limits:   &ResourceRequirements{CPUMillicores: 500, MemoryBytes: 134217728},
					},
					{
						Name:                   "fluentbit",
						Namespace:              "default",
						Kind:                   "DaemonSet",
						DesiredNumberScheduled: &desired,
						NumberReady:            &numReady,
						Containers: []ContainerInfoK8s{
							{Name: "fluentbit", Image: "fluent/fluent-bit:3.2"},
						},
					},
				},
				Services: []K8sServiceScanResult{
					{
						Name:      "nginx",
						Namespace: "default",
						Type:      "ClusterIP",
						ClusterIP: "10.43.100.1",
						Ports: []ServicePort{
							{Name: "http", Protocol: "TCP", Port: 80, TargetPort: "8080"},
						},
						Selector: map[string]string{"app": "nginx"},
					},
				},
				Ingresses: []IngressScanResult{
					{
						Name:         "nginx",
						Namespace:    "default",
						IngressClass: "traefik",
						Rules: []IngressRule{
							{
								Host: "nginx.example.com",
								Paths: []IngressPath{
									{Path: "/", Backend: "nginx", Port: "80"},
								},
							},
						},
						TLS: []IngressTLS{
							{Hosts: []string{"nginx.example.com"}, SecretName: "nginx-tls"},
						},
					},
				},
				ConfigMaps: []ConfigMapScanResult{
					{Name: "nginx-config", Namespace: "default", DataKeys: []string{"nginx.conf"}},
				},
				Secrets: []SecretScanResult{
					{Name: "nginx-tls", Namespace: "default", Type: "kubernetes.io/tls", DataKeys: []string{"tls.crt", "tls.key"}},
				},
				PVCs: []PVCScanResult{
					{Name: "data", Namespace: "default", StorageClass: "longhorn", AccessModes: []string{"ReadWriteOnce"}, Capacity: "10Gi", Status: "Bound"},
				},
				CronJobs: []CronJobScanResult{
					{Name: "backup", Namespace: "default", Schedule: "0 2 * * *", Suspend: false, LastScheduleTime: &lastSched},
				},
				NetworkPolicies: []NetworkPolicyScanResult{
					{Name: "deny-all", Namespace: "default", PodSelector: map[string]interface{}{"matchLabels": map[string]string{}}, PolicyTypes: []string{"Ingress", "Egress"}},
				},
				PDBs: []PDBScanResult{
					{Name: "nginx-pdb", Namespace: "default", MinAvailable: &minAvail, Selector: map[string]interface{}{"matchLabels": map[string]string{"app": "nginx"}}},
				},
			},
		},
		FluxDetected: true,
		FluxKustomizations: []FluxKustomizationResult{
			{
				Name:            "flux-system",
				Path:            "./clusters/k3s-ha",
				TargetNamespace: "",
				SourceRef:       map[string]interface{}{"kind": "GitRepository", "name": "flux-system"},
				Interval:        "10m0s",
				Prune:           true,
			},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Parse back as generic map to verify JSON keys
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Top-level keys
	for _, key := range []string{"name", "provider", "version", "nodes", "namespaces", "fluxDetected", "fluxKustomizations"} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing top-level key %q", key)
		}
	}

	// Node shape
	nodes := m["nodes"].([]interface{})
	node := nodes[0].(map[string]interface{})
	for _, key := range []string{"name", "status", "roles", "version", "os", "os_image"} {
		if _, ok := node[key]; !ok {
			t.Errorf("node missing key %q", key)
		}
	}

	// Namespace shape
	namespaces := m["namespaces"].([]interface{})
	ns := namespaces[0].(map[string]interface{})
	for _, key := range []string{"name", "labels", "workloads", "services", "ingresses", "configMaps", "secrets", "pvcs", "cronJobs", "networkPolicies", "pdbs"} {
		if _, ok := ns[key]; !ok {
			t.Errorf("namespace missing key %q", key)
		}
	}

	// Workload shape — deployment
	workloads := ns["workloads"].([]interface{})
	deploy := workloads[0].(map[string]interface{})
	for _, key := range []string{"name", "namespace", "kind", "replicas", "readyReplicas", "availableReplicas", "strategy", "containers", "requests", "limits"} {
		if _, ok := deploy[key]; !ok {
			t.Errorf("deployment workload missing key %q", key)
		}
	}

	// Workload shape — daemonset
	ds := workloads[1].(map[string]interface{})
	for _, key := range []string{"desiredNumberScheduled", "numberReady"} {
		if _, ok := ds[key]; !ok {
			t.Errorf("daemonset workload missing key %q", key)
		}
	}

	// Container shape
	containers := deploy["containers"].([]interface{})
	container := containers[0].(map[string]interface{})
	for _, key := range []string{"name", "image"} {
		if _, ok := container[key]; !ok {
			t.Errorf("container missing key %q", key)
		}
	}

	// Resource requirements shape
	req := deploy["requests"].(map[string]interface{})
	for _, key := range []string{"cpu_millicores", "memory_bytes"} {
		if _, ok := req[key]; !ok {
			t.Errorf("requests missing key %q", key)
		}
	}

	// Service shape
	services := ns["services"].([]interface{})
	svc := services[0].(map[string]interface{})
	for _, key := range []string{"name", "namespace", "type", "clusterIP", "ports", "selector"} {
		if _, ok := svc[key]; !ok {
			t.Errorf("service missing key %q", key)
		}
	}

	// Port shape
	ports := svc["ports"].([]interface{})
	port := ports[0].(map[string]interface{})
	for _, key := range []string{"name", "protocol", "port", "targetPort"} {
		if _, ok := port[key]; !ok {
			t.Errorf("port missing key %q", key)
		}
	}

	// Ingress shape
	ingresses := ns["ingresses"].([]interface{})
	ing := ingresses[0].(map[string]interface{})
	for _, key := range []string{"name", "namespace", "ingressClass", "rules", "tls"} {
		if _, ok := ing[key]; !ok {
			t.Errorf("ingress missing key %q", key)
		}
	}

	// Flux kustomization shape
	fluxKs := m["fluxKustomizations"].([]interface{})
	fk := fluxKs[0].(map[string]interface{})
	for _, key := range []string{"name", "path", "sourceRef", "interval", "prune"} {
		if _, ok := fk[key]; !ok {
			t.Errorf("flux kustomization missing key %q", key)
		}
	}
}

func TestExtractRoles(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected []string
	}{
		{
			name:     "control-plane and etcd",
			labels:   map[string]string{"node-role.kubernetes.io/control-plane": "", "node-role.kubernetes.io/etcd": ""},
			expected: []string{"control-plane", "etcd"},
		},
		{
			name:     "worker (no role labels)",
			labels:   map[string]string{"kubernetes.io/hostname": "worker-1"},
			expected: []string{"worker"},
		},
		{
			name:     "role with value",
			labels:   map[string]string{"node-role.kubernetes.io/": "infra"},
			expected: []string{"infra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles := extractRoles(tt.labels)
			if len(roles) == 0 {
				t.Fatal("expected at least one role")
			}
			// Check all expected roles are present (order may vary due to map iteration)
			roleSet := make(map[string]bool)
			for _, r := range roles {
				roleSet[r] = true
			}
			for _, exp := range tt.expected {
				if !roleSet[exp] {
					t.Errorf("expected role %q not found in %v", exp, roles)
				}
			}
		})
	}
}

func TestLabelSelectorToMap(t *testing.T) {
	sel := labelSelectorToMap(metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "nginx"},
	})
	if ml, ok := sel["matchLabels"].(map[string]string); !ok || ml["app"] != "nginx" {
		t.Errorf("expected matchLabels with app=nginx, got %v", sel)
	}
}

func TestOmitemptyBehavior(t *testing.T) {
	// Verify that empty optional fields are omitted from JSON
	result := ClusterScanResult{
		Name:    "test",
		Version: "v1.34.0",
		Nodes:   []NodeScanResult{},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	// provider and fluxDetected should be omitted when empty/false
	if _, ok := m["provider"]; ok {
		// provider is omitempty, so empty string should be omitted
		if m["provider"] == "" {
			t.Error("empty provider should be omitted")
		}
	}

	// fluxKustomizations should be omitted when nil
	if _, ok := m["fluxKustomizations"]; ok {
		t.Error("nil fluxKustomizations should be omitted")
	}

	// Workload with no optional fields
	w := WorkloadScanResult{
		Name:      "test",
		Namespace: "default",
		Kind:      "Deployment",
		Containers: []ContainerInfoK8s{
			{Name: "app", Image: "app:latest"},
		},
	}
	wData, _ := json.Marshal(w)
	var wm map[string]interface{}
	json.Unmarshal(wData, &wm)

	// replicas, readyReplicas, strategy should be omitted
	for _, key := range []string{"replicas", "readyReplicas", "availableReplicas", "strategy", "requests", "limits"} {
		if _, ok := wm[key]; ok {
			t.Errorf("optional field %q should be omitted when zero/nil", key)
		}
	}
}

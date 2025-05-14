package experiments

import "testing"

func TestRegistryGet(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	if _, err := registry.Get("kill-container"); err != nil {
		t.Fatalf("expected kill-container experiment: %v", err)
	}
	if _, err := registry.Get("net-latency"); err != nil {
		t.Fatalf("expected net-latency experiment: %v", err)
	}
	if _, err := registry.Get("cpu-stress"); err != nil {
		t.Fatalf("expected cpu-stress experiment: %v", err)
	}
	if _, err := registry.Get("unknown"); err == nil {
		t.Fatalf("expected error for unknown experiment")
	}
}

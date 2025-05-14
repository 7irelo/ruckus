package core

import "testing"

func TestNewRunID(t *testing.T) {
	t.Parallel()

	idA := NewRunID()
	idB := NewRunID()

	if idA == "" || idB == "" {
		t.Fatalf("run IDs must not be empty")
	}

	if idA == idB {
		t.Fatalf("run IDs must be unique, both were %q", idA)
	}

	if len(idA) < 5 || idA[:4] != "run-" {
		t.Fatalf("run ID must start with run-, got %q", idA)
	}
}

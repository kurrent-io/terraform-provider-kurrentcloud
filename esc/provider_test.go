package esc

import (
	"testing"
)

func TestProviderInternalValidate(t *testing.T) {
	if err := New("test")().InternalValidate(); err != nil {
		t.Fatalf("provider schema failed validation: %v", err)
	}
}

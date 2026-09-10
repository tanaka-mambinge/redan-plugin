package main

import "testing"

func TestDeleteIDRequiresConfirmationAndAcceptsFlag(t *testing.T) {
	if _, err := deleteID([]string{"delete", "fuel"}); err == nil {
		t.Fatal("expected confirmation to be required")
	}
	if id, err := deleteID([]string{"delete", "fuel", "--confirm"}); err != nil || id != "fuel" {
		t.Fatalf("deleteID = %q, %v", id, err)
	}
	if _, err := deleteID([]string{"delete", "fuel", "--force"}); err == nil {
		t.Fatal("expected unsupported flag to fail")
	}
}

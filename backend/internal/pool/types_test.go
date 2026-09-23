package pool

import (
	"context"
	"encoding/json"
	"testing"
)

// TestWorker_MarshalJSON_OmitsCancelFunc verifies that a Worker with a
// non-nil CancelFunc still marshals successfully, and that the func value
// itself is never included in the output.
func TestWorker_MarshalJSON_OmitsCancelFunc(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := Worker{
		ID:            1,
		WorkspaceID:   "workspace-a",
		WorkspaceName: "Workspace A",
		ActiveChange:  "add-feature",
		Status:        StatusWorking,
		CancelFunc:    cancel,
	}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("expected json.Marshal to succeed with a non-nil CancelFunc, got error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal encoded worker: %v", err)
	}

	if _, present := decoded["CancelFunc"]; present {
		t.Fatalf("expected CancelFunc to be omitted from marshaled output, got %s", data)
	}
	if _, present := decoded["cancel_func"]; present {
		t.Fatalf("expected CancelFunc to be omitted from marshaled output, got %s", data)
	}
}

package agents

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fakeAgy(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "agy")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0755); err != nil {
		t.Fatal(err)
	}
	old := agyBinary
	agyBinary = path
	ResetModelCache()
	t.Cleanup(func() { agyBinary = old; ResetModelCache() })
}

func TestAntigravityModels_Success(t *testing.T) {
	fakeAgy(t, "echo 'Fetching available models...'\nprintf 'm-a\\tModel A\\nm-b\\tModel B\\n'\n")
	a, _ := ByID("antigravity")
	got := a.Models(context.Background())
	if len(got) != 2 || got[0].ID != "m-a" || got[0].Label != "Model A" || got[0].Source != ModelSourceCLI {
		t.Fatalf("unexpected models: %v", got)
	}
}

func TestAntigravityModels_MergeDedupe(t *testing.T) {
	fakeAgy(t, "printf 'x\\tX\\nseed\\tSeed CLI\\n'\n")
	a, _ := ByID("antigravity")
	a.SeedModels = []Model{{ID: "seed", Label: "Seed", Source: ModelSourceSeed}}
	got := a.Models(context.Background())
	if len(got) != 2 || got[0].Source != ModelSourceSeed || got[1].ID != "x" {
		t.Fatalf("unexpected merge: %v", got)
	}
}

func TestAntigravityModels_FailureFallsBackToSeed(t *testing.T) {
	fakeAgy(t, "exit 1\n")
	a, _ := ByID("antigravity")
	a.SeedModels = []Model{{ID: "seed", Label: "Seed", Source: ModelSourceSeed}}
	got := a.Models(context.Background())
	if len(got) != 1 || got[0].ID != "seed" {
		t.Fatalf("expected seed fallback: %v", got)
	}
}

func TestAntigravityModels_Timeout(t *testing.T) {
	fakeAgy(t, "sleep 10\n")
	a, _ := ByID("antigravity")
	a.SeedModels = []Model{{ID: "seed", Label: "Seed", Source: ModelSourceSeed}}
	start := time.Now()
	got := a.Models(context.Background())
	if time.Since(start) > 6*time.Second {
		t.Fatalf("discovery did not time out: %v", time.Since(start))
	}
	if len(got) != 1 || got[0].ID != "seed" {
		t.Fatalf("expected seed fallback: %v", got)
	}
}

func TestModels_CachedAndSeedOnly(t *testing.T) {
	ResetModelCache()
	calls := 0
	a := AgentConfig{ID: "cached", ListModels: func(ctx context.Context) ([]Model, error) {
		calls++
		return nil, errors.New("boom")
	}}
	a.Models(context.Background())
	a.Models(context.Background())
	if calls != 1 {
		t.Errorf("expected cached discovery, got %d calls", calls)
	}
	ResetModelCache()
	gemini, _ := ByID("gemini")
	if got := gemini.Models(context.Background()); len(got) != len(gemini.SeedModels) {
		t.Errorf("seed only: %v", got)
	}
}

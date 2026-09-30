package agents

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	discoveryTimeout = 3 * time.Second
	discoveryTTL     = 5 * time.Minute
)

// agyBinary is the command used to discover Antigravity models; overridden in tests.
var agyBinary = "agy"

// listAntigravityModels runs `agy models` and parses "id<TAB>label" lines.
func listAntigravityModels(ctx context.Context) ([]Model, error) {
	ctx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, agyBinary, "models")
	cmd.WaitDelay = 200 * time.Millisecond // don't wait on pipes held by grandchildren
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseModelLines(out), nil
}

// parseModelLines parses tab-separated "id<TAB>label" lines; lines without a
// tab (progress messages) are ignored.
func parseModelLines(out []byte) []Model {
	var models []Model
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		id, label, ok := strings.Cut(sc.Text(), "\t")
		id = strings.TrimSpace(id)
		if !ok || id == "" {
			continue
		}
		label = strings.TrimSpace(label)
		if label == "" {
			label = id
		}
		models = append(models, Model{ID: id, Label: label, Source: ModelSourceCLI})
	}
	return models
}

type cacheEntry struct {
	models []Model
	at     time.Time
}

var (
	modelCacheMu sync.Mutex
	modelCache   = map[string]cacheEntry{}
)

// ResetModelCache clears the discovery cache (used by tests).
func ResetModelCache() {
	modelCacheMu.Lock()
	defer modelCacheMu.Unlock()
	modelCache = map[string]cacheEntry{}
}

// Models returns the seeded models merged without duplicates with the CLI
// discovery result (when the agent can list models). Discovery failures fall
// back to the seed list. Results are cached for a few minutes.
func (a AgentConfig) Models(ctx context.Context) []Model {
	seed := append([]Model(nil), a.SeedModels...)
	if a.ListModels == nil {
		return seed
	}
	modelCacheMu.Lock()
	if e, ok := modelCache[a.ID]; ok && time.Since(e.at) < discoveryTTL {
		modelCacheMu.Unlock()
		return append([]Model(nil), e.models...)
	}
	modelCacheMu.Unlock()

	discovered, err := a.ListModels(ctx)
	merged := seed
	if err == nil {
		seen := map[string]bool{}
		for _, m := range merged {
			seen[m.ID] = true
		}
		for _, m := range discovered {
			if !seen[m.ID] {
				seen[m.ID] = true
				merged = append(merged, m)
			}
		}
	}
	modelCacheMu.Lock()
	modelCache[a.ID] = cacheEntry{models: merged, at: time.Now()}
	modelCacheMu.Unlock()
	return append([]Model(nil), merged...)
}

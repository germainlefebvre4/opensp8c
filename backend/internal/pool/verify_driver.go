package pool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/glefebvre/opensp8c/internal/agents"
	"github.com/glefebvre/opensp8c/internal/verification"
)

// playwrightMCPVersion pins @playwright/mcp: an update of the tool must not
// silently change the behavior of an autonomous agent.
const playwrightMCPVersion = "0.0.83"

// readOnlyTools are always allowed next to the tools of a driver.
var readOnlyTools = []string{"Read", "Grep", "Glob"}

// permissionPromptsNone reports whether the installed claude CLI knows
// `--permission-prompts`; replaced by tests.
var permissionPromptsNone = func(ctx context.Context) bool {
	return agents.ClaudeSupportsFlag(ctx, "--permission-prompts")
}

// driverPlan is how the verifier agent is launched for a driver, computed
// without starting anything (see planDriver).
type driverPlan struct {
	Driver string
	// Args are the ExtraArgs of the agent.
	Args []string
	// AllowedTools is the list given to --allowedTools (empty for auto).
	AllowedTools []string
	// ToolPrefixes are the tool-name prefixes that count as using the driver.
	ToolPrefixes []string
	// Servers are the MCP servers expected `connected` in the init event.
	Servers []string
	// NeedChromeTools requires the Chrome integration tools in the init event.
	NeedChromeTools bool
	// Directive is the driver clause of the system prompt.
	Directive string
	// cleanup removes the temporary files of the plan; never nil.
	cleanup func()
}

const (
	chromeToolPrefix = "mcp__claude-in-chrome__"
	chromeToolsRule  = "mcp__claude-in-chrome"
)

var mcpNameUnsafe = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// mcpToolPrefix is the tool-name prefix of an MCP server (`mcp__<server>__`).
func mcpToolPrefix(server string) string {
	return "mcp__" + mcpNameUnsafe.ReplaceAllString(server, "_") + "__"
}

// planDriver checks what does not depend on the agent (npx, MCP file,
// allowed tools, agent type), writes the temporary Playwright MCP
// configuration outside the worktree, and returns the launch parameters of the
// driver. Nothing is started: every failure is testable without claude. The
// returned plan's cleanup must always be called.
func planDriver(ctx context.Context, res verification.Resolved, agent agents.AgentConfig, artifactsDir string) (driverPlan, error) {
	plan := driverPlan{Driver: res.UIDriver, cleanup: func() {}}
	if plan.Driver == "" {
		plan.Driver = verification.DriverAuto
	}

	if plan.Driver == verification.DriverAuto {
		if permissionPromptsNone(ctx) {
			plan.Args = []string{"--permission-prompts", "none"}
		}
		plan.Directive = autoDriverDirective
		return plan, nil
	}

	if !agent.SupportsDrivers() {
		return plan, fmt.Errorf("le pilote %s n'est supporté que par Claude (agent du rôle verifier : %s)", plan.Driver, agent.ID)
	}

	var mcpConfig string
	var tools []string
	switch plan.Driver {
	case verification.DriverPlaywright:
		if _, err := exec.LookPath("npx"); err != nil {
			return plan, fmt.Errorf("npx introuvable : le pilote playwright en a besoin (installer Node.js)")
		}
		path, cleanup, err := writePlaywrightConfig(artifactsDir)
		if err != nil {
			return plan, fmt.Errorf("impossible d'écrire la configuration MCP du pilote playwright : %w", err)
		}
		plan.cleanup = cleanup
		mcpConfig = path
		tools = []string{"mcp__playwright"}
		plan.Servers = []string{"playwright"}
		plan.ToolPrefixes = []string{mcpToolPrefix("playwright")}
		plan.Directive = driverDirective("Playwright")
	case verification.DriverChrome:
		tools = []string{chromeToolsRule}
		plan.NeedChromeTools = true
		plan.ToolPrefixes = []string{chromeToolPrefix}
		plan.Directive = driverDirective("the Chrome integration") + " " + chromeDirective
	case verification.DriverCustom:
		servers, err := checkCustomConfig(res.UIMcpConfig)
		if err != nil {
			return plan, err
		}
		if len(res.UIAllowedTools) == 0 {
			return plan, fmt.Errorf("uiAllowedTools non configuré")
		}
		mcpConfig = res.UIMcpConfig
		tools = res.UIAllowedTools
		plan.Servers = servers
		for _, s := range servers {
			plan.ToolPrefixes = append(plan.ToolPrefixes, mcpToolPrefix(s))
		}
		plan.Directive = driverDirective("the MCP servers you are given")
	default:
		return plan, fmt.Errorf("pilote inconnu : %s", plan.Driver)
	}

	plan.AllowedTools = append(append([]string(nil), tools...), readOnlyTools...)
	if permissionPromptsNone(ctx) {
		plan.Args = append(plan.Args, "--permission-prompts", "none")
	}
	if mcpConfig != "" {
		plan.Args = append(plan.Args, "--mcp-config", mcpConfig, "--strict-mcp-config")
	}
	if plan.NeedChromeTools {
		plan.Args = append(plan.Args, "--chrome")
	}
	plan.Args = append(plan.Args, "--allowedTools")
	plan.Args = append(plan.Args, plan.AllowedTools...)
	return plan, nil
}

// writePlaywrightConfig writes the MCP configuration of the platform's
// Playwright browser (headless, in-memory profile, captures and snapshots in
// outputDir) to a temporary directory outside any worktree.
func writePlaywrightConfig(outputDir string) (path string, cleanup func(), err error) {
	cfg := map[string]any{"mcpServers": map[string]any{"playwright": map[string]any{
		"command": "npx",
		"args": []string{"-y", "@playwright/mcp@" + playwrightMCPVersion,
			"--headless", "--isolated", "--output-dir", outputDir},
	}}}
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", nil, err
	}
	dir, err := os.MkdirTemp("", "opensp8c-mcp-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(dir) }
	path = filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
}

// checkCustomConfig reads the user's MCP file and returns its server names.
func checkCustomConfig(path string) ([]string, error) {
	if path == "" {
		return nil, fmt.Errorf("uiMcpConfig non configuré")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fichier uiMcpConfig illisible (%s) : %v", path, err)
	}
	var cfg struct {
		McpServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("fichier uiMcpConfig %s : JSON invalide : %v", path, err)
	}
	if len(cfg.McpServers) == 0 {
		return nil, fmt.Errorf("fichier uiMcpConfig %s : aucun serveur dans mcpServers", path)
	}
	names := make([]string, 0, len(cfg.McpServers))
	for n := range cfg.McpServers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

// Driver clauses of the system prompt.
const (
	autoDriverDirective = "Choose the way you drive the browser among what you have (a browser tool of your host, or a throwaway script, for example Playwright through npx, kept outside the repository)."
	chromeDirective     = "Open only your own tabs: never read, navigate or modify a tab the user already has open."
)

func driverDirective(name string) string {
	return "Drive the browser only with the tools of " + name + "; run no script and no command."
}

// errInitUnreadable is the reason when the init event lacks what the driver
// check needs: failing closed beats a blind PASS.
const errInitUnreadable = "événement d'initialisation illisible"

// driverObserver checks the init event of the agent's stream against the plan
// and counts the calls to the driver's tools. Not safe for concurrent use: it
// is fed by the single goroutine that reads the turn.
type driverObserver struct {
	plan     driverPlan
	initSeen bool
	calls    int
}

func newDriverObserver(plan driverPlan) *driverObserver { return &driverObserver{plan: plan} }

// Observe is a turnTarget.observe: it returns an error when the init event
// shows an unusable driver.
func (o *driverObserver) Observe(line []byte) error {
	if o.plan.Driver == verification.DriverAuto {
		return nil
	}
	switch {
	case bytes.Contains(line, []byte(`"init"`)):
		return o.observeInit(line)
	case bytes.Contains(line, []byte(`"tool_use"`)):
		o.observeToolUse(line)
	}
	return nil
}

// Calls is the number of calls to a tool of the driver seen so far.
func (o *driverObserver) Calls() int { return o.calls }

// InitSeen reports whether the init event was seen.
func (o *driverObserver) InitSeen() bool { return o.initSeen }

func (o *driverObserver) observeInit(line []byte) error {
	var ev struct {
		Type       string `json:"type"`
		Subtype    string `json:"subtype"`
		McpServers *[]struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"mcp_servers"`
		Tools *[]string `json:"tools"`
	}
	if json.Unmarshal(line, &ev) != nil || ev.Type != "system" || ev.Subtype != "init" {
		return nil // another event that merely mentions "init"
	}
	o.initSeen = true
	if len(o.plan.Servers) > 0 {
		if ev.McpServers == nil {
			return errors.New(errInitUnreadable)
		}
		status := map[string]string{}
		for _, s := range *ev.McpServers {
			status[s.Name] = s.Status
		}
		for _, name := range o.plan.Servers {
			st, ok := status[name]
			switch {
			case !ok:
				return fmt.Errorf("serveur MCP « %s » absent de l'événement d'initialisation de l'agent", name)
			case st != "connected":
				return fmt.Errorf("serveur MCP « %s » non connecté (statut : %s)", name, st)
			}
		}
	}
	if o.plan.NeedChromeTools {
		if ev.Tools == nil {
			return errors.New(errInitUnreadable)
		}
		for _, t := range *ev.Tools {
			if strings.HasPrefix(t, chromeToolPrefix) {
				return nil
			}
		}
		return fmt.Errorf("aucun outil de l'intégration Chrome n'est disponible : vérifier que Chrome est ouvert avec l'extension connectée")
	}
	return nil
}

func (o *driverObserver) observeToolUse(line []byte) {
	var ev struct {
		Type    string `json:"type"`
		Message struct {
			Content []struct {
				Type string `json:"type"`
				Name string `json:"name"`
			} `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &ev) != nil || ev.Type != "assistant" {
		return
	}
	for _, b := range ev.Message.Content {
		if b.Type != "tool_use" {
			continue
		}
		for _, p := range o.plan.ToolPrefixes {
			if strings.HasPrefix(b.Name, p) {
				o.calls++
				break
			}
		}
	}
}

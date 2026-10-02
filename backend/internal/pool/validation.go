package pool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/glefebvre/opensp8c/internal/session"
)

// validationTimeout bounds one validation command; validationWaitDelay is how
// long Wait may block on inherited pipes after the command is killed. Variables
// so tests can shorten them.
var (
	validationTimeout   = 20 * time.Minute
	validationWaitDelay = 5 * time.Second
)

// ValidationEnvError reports that validation could not run for an
// environment reason (no command, missing executable or directory), as opposed
// to a failing test. An agent heal turn cannot fix it, so the worker pauses.
type ValidationEnvError struct{ Reason string }

func (e *ValidationEnvError) Error() string { return e.Reason }

// noValidationReason is the pause reason when nothing is configured or detected.
const noValidationReason = "Aucune commande de validation détectée (ni go.mod, ni package.json avec script « test » et node_modules) : configurez une commande de validation dans les réglages du pool (par exemple « true » pour désactiver la validation)."

// validationCommand is one detected command and the directory it runs in.
type validationCommand struct {
	Dir  string
	Name string
	Args []string
}

func (c validationCommand) String() string {
	return strings.Join(append([]string{c.Name}, c.Args...), " ")
}

// DetectValidationCommands inspects root, then its direct subdirectories
// (hidden ones and node_modules excluded, alphabetical order), and returns
// `go test ./...` for each go.mod and `npm test` for each package.json that
// declares a test script and has its dependencies installed (node_modules).
func DetectValidationCommands(root string) []validationCommand {
	cmds, _ := detectValidation(root)
	return cmds
}

// detectValidation is DetectValidationCommands plus the directories (relative
// to root, "." for the root itself) holding a package.json test script whose
// dependencies are not installed: detected but not validable.
func detectValidation(root string) ([]validationCommand, []string) {
	dirs := []string{root}
	if entries, err := os.ReadDir(root); err == nil {
		for _, e := range entries {
			name := e.Name()
			if !e.IsDir() || strings.HasPrefix(name, ".") || name == "node_modules" {
				continue
			}
			dirs = append(dirs, filepath.Join(root, name))
		}
	}

	var cmds []validationCommand
	var unvalidable []string
	for _, dir := range dirs {
		if fileExists(filepath.Join(dir, "go.mod")) {
			cmds = append(cmds, validationCommand{Dir: dir, Name: "go", Args: []string{"test", "./..."}})
		}
		if hasTestScript(filepath.Join(dir, "package.json")) {
			if dirExists(filepath.Join(dir, "node_modules")) {
				cmds = append(cmds, validationCommand{Dir: dir, Name: "npm", Args: []string{"test"}})
			} else {
				rel, err := filepath.Rel(root, dir)
				if err != nil {
					rel = dir
				}
				unvalidable = append(unvalidable, rel)
			}
		}
	}
	return cmds, unvalidable
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func hasTestScript(packageJSON string) bool {
	data, err := os.ReadFile(packageJSON)
	if err != nil {
		return false
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}
	return strings.TrimSpace(pkg.Scripts["test"]) != ""
}

// runValidation resolves the validation command at call time (configured
// command, else auto-detection) and runs it in the worker's worktree. It
// returns a *ValidationEnvError when validation cannot run at all, and an
// ordinary error when a started command fails.
func (m *Manager) runValidation(ctx context.Context, w *Worker) error {
	return m.runValidationIn(ctx, w.WorkspaceID, w.WorktreePath)
}

// runValidationIn is runValidation for a workspace and a directory, without
// requiring a Worker (used by the review approval).
func (m *Manager) runValidationIn(ctx context.Context, workspaceID, dir string) error {
	configured := ""
	if m.prefs != nil {
		if p, err := m.prefs.Load(); err == nil && p != nil {
			configured = strings.TrimSpace(p.ResolvePool(workspaceID).ValidationCommand)
		}
	}

	if configured != "" {
		_, err := runValidationCommand(ctx, dir, "sh", []string{"-c", configured}, configured, true)
		return err
	}

	cmds, unvalidable := detectValidation(dir)
	if len(unvalidable) > 0 {
		return &ValidationEnvError{Reason: fmt.Sprintf("Projet détecté mais non validable faute de dépendances installées (script « test » sans node_modules) : %s. Configurez une commande de validation dans les réglages du pool qui installe les dépendances (par exemple « npm ci && npm test »).", strings.Join(unvalidable, ", "))}
	}
	if len(cmds) == 0 {
		return &ValidationEnvError{Reason: noValidationReason}
	}
	var combined strings.Builder
	for _, c := range cmds {
		out, err := runValidationCommand(ctx, c.Dir, c.Name, c.Args, c.String(), false)
		combined.WriteString(out)
		if err != nil {
			var envErr *ValidationEnvError
			if errors.As(err, &envErr) {
				return err
			}
			return fmt.Errorf("%v\nOutput:\n%s", err, combined.String())
		}
	}
	return nil
}

// runValidationCommand runs one command in dir. viaShell marks `sh -c`
// commands, where exit status 127 means the command was not found.
func runValidationCommand(ctx context.Context, dir, name string, args []string, display string, viaShell bool) (string, error) {
	if !dirExists(dir) {
		return "", &ValidationEnvError{Reason: fmt.Sprintf("Répertoire d'exécution de la validation introuvable : %s", dir)}
	}
	tctx, cancel := context.WithTimeout(ctx, validationTimeout)
	defer cancel()
	cmd := exec.CommandContext(tctx, name, args...)
	cmd.Dir = dir
	// Own process group, killed whole on cancellation or timeout; WaitDelay
	// unblocks Wait when a grandchild keeps the output pipe open.
	session.ApplyProcessGroup(cmd)
	cmd.WaitDelay = validationWaitDelay
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), nil
	}
	if ctx.Err() == nil && errors.Is(tctx.Err(), context.DeadlineExceeded) {
		return string(out), &ValidationEnvError{Reason: fmt.Sprintf("Validation trop longue : la commande « %s » a dépassé le délai de %s et a été arrêtée.", display, validationTimeout)}
	}
	if ctx.Err() == nil {
		if errors.Is(err, exec.ErrNotFound) {
			return string(out), &ValidationEnvError{Reason: fmt.Sprintf("Commande de validation introuvable : %s", display)}
		}
		var exitErr *exec.ExitError
		if viaShell && errors.As(err, &exitErr) && exitErr.ExitCode() == 127 {
			return string(out), &ValidationEnvError{Reason: fmt.Sprintf("Commande de validation introuvable : %s\n%s", display, strings.TrimSpace(string(out)))}
		}
	}
	return string(out), fmt.Errorf("tests failed (%s in %s): %v\nOutput:\n%s", display, dir, err, string(out))
}

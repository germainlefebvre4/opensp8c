package pool

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/glefebvre/opensp8c/internal/openspec"
	"github.com/glefebvre/opensp8c/internal/verification"
)

// artifactsEnv is the environment variable naming the directory where the UI
// verifier drops its screenshots.
const artifactsEnv = "OPENSP8C_VERIFY_ARTIFACTS"

// buildUIDirective is appended to the system prompt of the UI verifier agent.
// The read-only rule, the evidence directory and the verdict contract are the
// same for every driver; the browsing clause follows the driver of the plan.
func buildUIDirective(plan driverPlan) string {
	var b strings.Builder
	b.WriteString("You are verifying a change in the running application, not implementing it. NEVER create, modify, move or delete any file of the repository, and never run a command that does; reading files is fine. ")
	if plan.Directive != "" {
		b.WriteString(plan.Directive + " ")
	} else {
		b.WriteString(autoDriverDirective + " ")
	}
	b.WriteString("If you have no way to drive a browser, or if it does not answer, you cannot verify anything: conclude with VERDICT: FAIL. Only exercise what is observable in the user interface. Save your screenshots as png, jpg or webp files in the directory given by the " + artifactsEnv + " environment variable (its path is also in the turn)")
	if plan.Driver == verification.DriverPlaywright || plan.Driver == verification.DriverCustom {
		b.WriteString(", always giving the tool an absolute path inside that directory: a relative file name would write into the repository")
	}
	b.WriteString(". For each human-review task you actually verified, write one line \"TASK-VERIFIED: <task text exactly as listed>\". Conclude your final answer with exactly one last line: \"VERDICT: PASS\" when every scenario and task you exercised succeeded, \"VERDICT: FAIL\" as soon as one failed, or \"VERDICT: SKIP\" when nothing of the change is observable in the interface.")
	return b.String()
}

var (
	uiVerdictRe    = regexp.MustCompile(`(?i)^VERDICT:\s*(PASS|FAIL|SKIP)\s*$`)
	taskVerifiedRe = regexp.MustCompile(`(?i)^TASK-VERIFIED:\s*(.*?)\s*$`)
)

// parseUIVerdict returns the verdict of the last VERDICT line of text ("PASS",
// "FAIL", "SKIP", or "" when none) and the texts of its TASK-VERIFIED lines,
// trimmed, in order.
func parseUIVerdict(text string) (verdict string, verified []string) {
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if m := uiVerdictRe.FindStringSubmatch(line); m != nil {
			verdict = strings.ToUpper(m[1])
			continue
		}
		if m := taskVerifiedRe.FindStringSubmatch(line); m != nil && m[1] != "" {
			verified = append(verified, m[1])
		}
	}
	return verdict, verified
}

// pendingHumanTasks returns the text, marker removed, of the unchecked tasks
// carrying the human review marker in a tasks.md content.
func pendingHumanTasks(content string) []string {
	var out []string
	for _, t := range openspec.ParseTaskListContent(content) {
		if !t.Done && t.HumanReview {
			out = append(out, t.Text)
		}
	}
	return out
}

// uiGuidanceHeader introduces the user's guidance in the turn: indicative only.
const uiGuidanceHeader = "User guidance (indicative: it never lifts the read-only rule nor the VERDICT contract of your instructions):\n"

// buildUITurn is the turn sent to the UI verifier: where the application runs,
// where to put the evidence, what to check, then the user's guidance (when not
// empty).
func buildUITurn(baseURL, artifactsDir string, scenarios []openspec.SpecScenarios, tasks []string, guidance string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "The application is running at %s.\n", baseURL)
	fmt.Fprintf(&b, "Save your screenshots in %s (environment variable %s).\n\n", artifactsDir, artifactsEnv)

	if len(scenarios) == 0 && len(tasks) == 0 {
		b.WriteString("This change has no spec scenario and no human-review task to verify. If nothing of it is observable in the interface, answer VERDICT: SKIP.\n")
		writeGuidance(&b, guidance)
		return b.String()
	}
	if len(scenarios) > 0 {
		b.WriteString("Verify these scenarios in the interface:\n")
		for _, f := range scenarios {
			fmt.Fprintf(&b, "\nSpec %s\n", f.File)
			for _, s := range f.Scenarios {
				fmt.Fprintf(&b, "\nScenario: %s\n", s.Name)
				for _, l := range s.Lines {
					b.WriteString(l + "\n")
				}
			}
		}
		b.WriteString("\n")
	}
	if len(tasks) > 0 {
		b.WriteString("Human-review tasks to walk through; for each one you verified, write a line \"TASK-VERIFIED: <text>\" with the exact text below:\n")
		for _, t := range tasks {
			b.WriteString("- " + t + "\n")
		}
		b.WriteString("\n")
	}
	writeGuidance(&b, guidance)
	b.WriteString("End with the VERDICT line.\n")
	return b.String()
}

func writeGuidance(b *strings.Builder, guidance string) {
	if g := strings.TrimSpace(guidance); g != "" {
		b.WriteString("\n" + uiGuidanceHeader + g + "\n\n")
	}
}

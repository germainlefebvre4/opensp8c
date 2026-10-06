package pool

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// maxHeaderLen bounds the first line of a commit message.
const maxHeaderLen = 72

// commitTypes maps the first word of a change name to its Conventional
// Commits type; any other word gives "feat".
var commitTypes = map[string]string{
	"fix":      "fix",
	"refactor": "refactor",
	"perf":     "perf",
	"docs":     "docs",
	"doc":      "docs",
	"test":     "test",
	"tests":    "test",
	"chore":    "chore",
	"ci":       "ci",
	"build":    "build",
	"style":    "style",
}

func splitWords(name string) []string {
	return strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' || unicode.IsSpace(r) })
}

// commitType deduces the commit type from the first word of a change name.
func commitType(name string) string {
	words := splitWords(name)
	if len(words) == 0 {
		return "feat"
	}
	if t, ok := commitTypes[strings.ToLower(words[0])]; ok {
		return t
	}
	return "feat"
}

// commitSubject formats a change name as a sentence: dashes and underscores
// become spaces and the first letter is upper-cased.
func commitSubject(name string) string {
	s := strings.Join(splitWords(name), " ")
	r, size := utf8.DecodeRuneInString(s)
	if size == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

// normalizeScope lower-cases a scope and replaces every character outside
// [a-z0-9._/-] by a dash, merging and trimming dashes. An empty result means
// "no scope".
func normalizeScope(raw string) string {
	var b strings.Builder
	lastDash := true // drops leading dashes
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '/'
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// commitHeader assembles "type(scope): subject", or "type: subject" without
// scope, truncating the subject on a word boundary so that the header fits in
// maxHeaderLen characters.
func commitHeader(typ, scope, subject string) string {
	prefix := typ + ": "
	if scope != "" {
		prefix = typ + "(" + scope + "): "
	}
	room := maxHeaderLen - utf8.RuneCountInString(prefix)
	if utf8.RuneCountInString(subject) > room {
		cut := []rune(subject)
		if room < 0 {
			room = 0
		}
		cut = cut[:room]
		trimmed := strings.TrimRightFunc(string(cut), unicode.IsSpace)
		// Cut on a word boundary unless the subject is a single long word or
		// the cut already falls exactly between two words.
		if next := []rune(subject)[room]; !unicode.IsSpace(next) {
			if i := strings.LastIndexFunc(trimmed, unicode.IsSpace); i >= 0 {
				trimmed = strings.TrimRightFunc(trimmed[:i], unicode.IsSpace)
			}
		}
		subject = trimmed
	}
	return prefix + subject
}

// changeCommitMessage builds the full message of a commit made for a change:
// the Conventional Commits header, a blank line and the "Change:" body.
func changeCommitMessage(name, scope string) string {
	return buildMessage(commitType(name), scope, commitSubject(name), name)
}

// correctionCommitMessage is the message of a review correction commit.
func correctionCommitMessage(name, scope string) string {
	return buildMessage("chore", scope, "Add review correction", name)
}

func buildMessage(typ, scope, subject, name string) string {
	return commitHeader(typ, normalizeScope(scope), subject) + "\n\nChange: " + name
}

// changeScope deduces the commit scope of a change: the first non-empty
// tags.components of its .openspec.yaml, else the first capability folder of
// its specs/ (alphabetical), else "". It reads the change folder of the main
// repository, then the one of the worktree, and never reports an error.
func (wc *WorktreeController) changeScope(name string) string {
	dirs := []string{
		filepath.Join(wc.repoRoot, "openspec", "changes", name),
		filepath.Join(wc.resolvePath(name), "openspec", "changes", name),
	}
	for _, dir := range dirs {
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			continue
		}
		if scope := scopeFromDir(dir); scope != "" {
			return scope
		}
	}
	return ""
}

func scopeFromDir(dir string) string {
	if data, err := os.ReadFile(filepath.Join(dir, ".openspec.yaml")); err == nil {
		var meta struct {
			Tags *struct {
				Components []string `yaml:"components"`
			} `yaml:"tags"`
		}
		if yaml.Unmarshal(data, &meta) == nil && meta.Tags != nil {
			for _, c := range meta.Tags.Components {
				if s := normalizeScope(c); s != "" {
					return s
				}
			}
		}
	}
	entries, err := os.ReadDir(filepath.Join(dir, "specs"))
	if err != nil {
		return ""
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		if s := normalizeScope(n); s != "" {
			return s
		}
	}
	return ""
}

// commitMessageFor is the message of the worker commit and of the merge of a
// change.
func (wc *WorktreeController) commitMessageFor(name string) string {
	return changeCommitMessage(name, wc.changeScope(name))
}

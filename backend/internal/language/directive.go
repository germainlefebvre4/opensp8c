package language

import "strings"

// Role designates the kind of agent a directive is built for.
type Role int

const (
	// Chat: exploration sessions (conversation language).
	Chat Role = iota
	// Docs: documentation generation, fast-forward, exploration promotion.
	Docs
	// Worker: pool worker implementing a change (code + documentation).
	Worker
)

func name(code string) string {
	if l, ok := Lookup(code); ok {
		return l.EnglishName
	}
	return code
}

const defaultClause = "Unless the user explicitly asks otherwise, or the project's conventions (for example an existing i18n mechanism) require it, "

const formalismClause = "This applies to prose only: keep OpenSpec structural headings (### Requirement:, #### Scenario:), the normative keywords SHALL/MUST, the WHEN/THEN markers and mandated file names unchanged."

// Directive builds the language instruction for a role from resolved languages.
func Directive(role Role, r Resolved) string {
	var b strings.Builder
	switch role {
	case Chat:
		b.WriteString(defaultClause + "converse with the user in " + name(r.Chat) + ".")
	case Docs:
		b.WriteString(defaultClause + "write documentation and OpenSpec artifacts (proposal, design, specs, tasks) in " + name(r.Documentation) + ". " + formalismClause)
	case Worker:
		b.WriteString(defaultClause + "write everything that goes into the code in " + name(r.Code) +
			": identifiers, comments, commit messages, error messages and user-visible text. ")
		b.WriteString("Write OpenSpec files and documentation you modify in " + name(r.Documentation) + ". " + formalismClause)
	}
	return b.String()
}

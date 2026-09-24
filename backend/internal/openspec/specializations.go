package openspec

// BaseAgentSpecializations is the fixed, hard-coded vocabulary of agent
// specialization tags, generic and independent of any workspace's tech stack.
var BaseAgentSpecializations = []string{
	"frontend",
	"backend",
	"database",
	"api-design",
	"devops",
	"testing",
	"security",
	"documentation",
	"ux-design",
	"data",
	"mobile",
}

// CombineSpecializationVocabulary merges the base vocabulary with user-defined
// extensions, deduplicating so a value already in the base is never repeated.
func CombineSpecializationVocabulary(custom []string) []string {
	seen := make(map[string]struct{}, len(BaseAgentSpecializations))
	combined := make([]string, 0, len(BaseAgentSpecializations)+len(custom))

	for _, v := range BaseAgentSpecializations {
		seen[v] = struct{}{}
		combined = append(combined, v)
	}
	for _, v := range custom {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		combined = append(combined, v)
	}
	return combined
}

// SanitizeCustomSpecializations strips out any value already present in the base
// vocabulary and deduplicates the remaining values, so a custom extension never
// shadows or repeats a base value once persisted.
func SanitizeCustomSpecializations(custom []string) []string {
	base := make(map[string]struct{}, len(BaseAgentSpecializations))
	for _, v := range BaseAgentSpecializations {
		base[v] = struct{}{}
	}

	seen := make(map[string]struct{}, len(custom))
	sanitized := make([]string, 0, len(custom))
	for _, v := range custom {
		if _, inBase := base[v]; inBase {
			continue
		}
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		sanitized = append(sanitized, v)
	}
	return sanitized
}

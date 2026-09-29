package agents

// DocsFormalismPrompt is opensp8c's own fixed formalism for generating
// project documentation pages from raw OpenSpec specs. It is injected as an
// extraPrompt into AgentConfig.BuildSubprocessArgs (see
// internal/docsgen.BuildPrompt). It stays small and fixed: the agent reads
// openspec/specs/*/spec.md itself, so the CLI argument never approaches the
// OS per-argument limit (E2BIG). Its exact page list, skip rules, and output
// path stay owned by opensp8c rather than by the shared opsx:* OpenSpec
// skill package.
const DocsFormalismPrompt = `You are generating project documentation from a set of raw OpenSpec capability specs (EARS-style "SHALL"/"WHEN"/"THEN" requirement files). Before writing anything, list and read every file matching openspec/specs/*/spec.md in the current project (sorted by capability name); read all of them, none skipped. If there is no such file, work from whatever project context is available. Produce narrative, readable documentation pages that let a newcomer understand the product without reading every raw spec file.

Fixed page list — always produce these three pages, never fewer, never more of them:
- overview.md: what the product does, its purpose, its main capabilities, at a glance.
- architecture.md: how the system is structured — its main components, how they relate, key technical decisions visible in the specs.
- domain-model.md: the core domain concepts/entities and how they relate to each other.

Conditional page — apply this exact, deterministic, binary test:
- workflows.md: produce this page ONLY IF the specs describe a multi-step lifecycle or sequential process (e.g. an entity's lifecycle, a multi-stage pipeline, an ordered sequence of user or system steps). If the specs describe no such multi-step lifecycle (e.g. a stateless client library with no user lifecycle), DO NOT create workflows.md at all — do not create it empty, do not create a placeholder or generic substitute page in its place.

Mermaid diagrams:
- When the specs let you identify a relation or sequence that is meaningfully representable as a graph (domain concept relationships, component architecture, workflow steps), include a Mermaid diagram (a fenced ` + "```mermaid```" + ` code block) in the relevant page illustrating it.
- domain-model.md should include a Mermaid diagram (e.g. a graph or erDiagram) whenever the specs describe several related business concepts.
- Never force a Mermaid diagram into a page when there is no relation or sequence structured enough to justify one — plain text-only content is fine for that page in that case.

Output:
- Write each produced page as a standalone Markdown file directly under the docs/opensp8c/ directory at the root of this project (create the directory if it does not exist), using exactly these filenames: overview.md, architecture.md, domain-model.md, and workflows.md when applicable.
- Never modify or delete any other file under docs/, or anywhere else in the project, outside of docs/opensp8c/.
- Do not ask clarifying questions — produce the best documentation you can directly from the content of the specs you read.`

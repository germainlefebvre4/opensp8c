# opensp8c — Overview

## What it is

**opensp8c** is a local web application that puts a visual, collaborative layer on top of [OpenSpec](https://github.com/Fission-AI/OpenSpec) projects. It reads the `openspec/` folder of one or more *workspaces* (project directories) and turns OpenSpec **changes** into cards on a Kanban board. From that board a user can explore an idea with an AI agent, generate the change artifacts (proposal, design, tasks), have a pool of autonomous agents implement it, review the result, and archive it.

It is a Go backend serving a React single-page frontend (embedded in the binary), talking to external agent CLIs (Claude, Codex, Gemini, Antigravity, Copilot) and to the `openspec` CLI.

## Purpose

- Make the OpenSpec lifecycle (explore → specify → implement → review → archive) visible and drivable from a UI.
- Let AI agents do the work — chat-based exploration, artifact generation ("fast-forward"), parallel implementation in isolated git worktrees — while keeping the human in control.
- Give a navigable view of the project's history: specs, the changes that touched them, and generated documentation.

## Main capabilities

| Area | What the user gets |
|---|---|
| **Workspaces** | Add/remove project directories (must contain `openspec/`), switch the active one, see per-status counters in the sidebar; the active workspace is kept in the URL (`?workspace=<id>`). |
| **Kanban board** | Six slots: *To Explore*, *Ready*, *To Do*, *In Progress*, *To Review*, *Done/Archived*. Search, drag-and-drop with a strict transition matrix, stale-change badges, live refresh through SSE. |
| **Exploration** | Chat with an agent from a bottom panel, anonymous or attached to a change. Streaming, tool-call display, structured question cards, "ghost cards" for explorations not yet promoted, split-screen task drafts, session resume. |
| **Change details** | Side panel with tasks (toggleable), proposal, design, tags, actions (delete, archive, approve/request corrections) and a read-only *Conversation* activity feed with a colored timeline. |
| **Agent pool** | Per-workspace pool (1–5 workers) executing *To Do* changes in parallel from a dependency DAG, each in its own git worktree, with a self-healing loop and two delegation modes (`full-autonomy`, `hitl-review`). |
| **Review & archive** | HITL review panel with diff, approve-and-merge or request corrections; "Sync & Archive" runs `openspec archive`. |
| **Tags & staleness** | Automatic semantic tags (type, complexity, components, agent specialization) and stale detection based on `tasks.md` activity. |
| **Specs & history** | Spec browser with TOC and inline editor (live diff), Timeline of all changes with filters and heatmap, spec × time matrix. |
| **Generated documentation** | One agent run produces `docs/opensp8c/` pages (overview, architecture, domain model, optional workflows) with Mermaid diagrams. |
| **Platform configuration** | Agent registry and detection, default agent, global and per-agent environment variables, native question mode, UI language and per-purpose agent languages (chat / documentation / code). |
| **i18n** | English and French UI, every string behind a translation key. |
| **Developer tooling** | Makefile targets, multi-stage Docker image, docker-compose, `--port`/`--host` flags. |

## Reading guide

- [architecture.md](architecture.md) — components, storage, real-time channels, agent integration.
- [domain-model.md](domain-model.md) — the core concepts and how they relate.
- [workflows.md](workflows.md) — the change lifecycle, exploration, pool execution, review and other sequences.

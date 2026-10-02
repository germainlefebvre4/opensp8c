# OpenSp8c

## Getting Started

## Agent CLI Usage

You can pass environment variables in your terminal to configure the agent of opensp8c.

### Gemini

Useful environment variables for Gemini:

* `GEMINI_MODEL`: The model to use for Gemini.
* `GOOGLE_CLOUD_PROJECT`: The Google Cloud project ID related to your Gemini Enterprise account.

## Agent Pool

The agent pool implements launched changes in isolated git worktrees, one branch `feature/<change>` per change.

### Worktree location

Worktrees live in `~/.opensp8c/worktrees/<workspaceID>/wt-<change>`, so two workspaces with a change of the same name never share a directory. Worktrees created at the former location (`~/.opensp8c/worktrees/wt-<change>`) are reused as-is, with their uncommitted changes.

Set `OPENSP8C_WORKTREES_DIR` to use another root directory (read once when the pool manager is created):

```
OPENSP8C_WORKTREES_DIR=/tmp/wt make dev-backend
```

### A change must be committed before it is launched

A worktree starts from the last commit of the repository's current branch: a change whose files were never committed is absent from it. The worker then does not start any agent and pauses with this reason:

> Le changement « <change> » doit être committé dans le dépôt avant d'être lancé (openspec/changes/<change>/tasks.md est absent du worktree).

Commit the change (`git add openspec/changes/<change> && git commit`), then resume the worker.

Nothing is created in that case (no branch, no worktree), so committing and resuming is all it takes.

If the branch `feature/<change>` already exists but was created before the change was committed, the worker handles it by itself when the branch holds no work (clean worktree, no commit of its own): the branch and its worktree are recreated from the current branch, without forcing anything. When the branch does hold work, nothing is deleted or rewritten and the worker pauses with a distinct reason:

> La branche « feature/<change> » ne contient pas le changement (créée avant son commit) : y intégrer la branche courante ou la supprimer, puis reprendre le worker.

Merge the current branch into `feature/<change>` (or delete the branch if its work is not needed), then resume the worker.

### Finalization

When validation passes and every task of `tasks.md` is checked, the worker commits the agent's work in `feature/<change>`, then:

* `full-autonomy`: merges the branch into the repository's current branch and only then removes the worktree and the branch. On a conflict or an already running merge, the merge is aborted (your own merge is never touched), the branch and the worktree are kept and the worker pauses with the reason.
* `hitl-review`: keeps the branch and the worktree and puts the change in To Review. The change is not dispatched again and stays in To Review, even after the pool or the backend restarts, until you act on it (see below).

### Reviewing a change

A change in To Review has a Review tab in its detail panel, listing the files and diff of `feature/<change>`. Two actions are available, from the Actions tab or by dragging the card:

* **Approve & Merge** (drop on Done): after a confirmation, merges the branch into the repository's current branch (integrating and revalidating first if that branch advanced), then removes the worktree and the branch. It works with the pool stopped.
* **Request a correction** (drop on In Progress): adds your feedback as an unchecked task in a `## Corrections` section of the change's `tasks.md`, commits it in `feature/<change>` and puts the change back in the queue.

If the merge succeeded but the worktree or the branch could not be removed (for example an untracked file left in the worktree), the change still moves to Done and a warning toast lists what is left to clean by hand: remove the worktree with `git worktree remove <path>` (add `--force` to discard its untracked files), then delete the branch with `git branch -d feature/<change>`. A branch left behind would be reused if a change with the same name is created again.

A worker also pauses, with a readable reason, when the agent ends a turn with an error result, stays silent for 30 minutes, or when the validation command runs for more than 20 minutes. Agent and validation processes run in their own process group, which is killed as a whole on cancellation, pool stop and server shutdown.

### Using `full-autonomy` safely

In `full-autonomy` the validation command is the only barrier before the work reaches your branch:

* **Node projects need an explicit validation command.** A fresh worktree has no `node_modules`. When the auto-detection finds a `package.json` with a `test` script but no installed dependencies, the worker pauses immediately (no command runs, no heal turn) and names the directory. Set a validation command in the pool settings that installs the dependencies, for example for a Go backend plus a Node frontend (run with `sh -c` at the worktree root):

  ```
  cd backend && go test ./... && cd ../frontend && npm ci && npm test
  ```

* **The merge only happens in the branch the change started from.** The branch is recorded when `feature/<change>` is created. If the repository is on another branch (or on a detached HEAD) when the worker is ready to merge, it pauses and names both branches; check the base branch out again and resume the worker.
* **Commits that landed on the target meanwhile are integrated and revalidated.** If the current branch received commits since the change started, the worker merges them into `feature/<change>`, runs the validation again (with the same `max_attempts` heal budget) and only then merges. An integration conflict is aborted and pauses the worker. If the target keeps moving, the worker retries up to 3 integrations before pausing.
* **Avoid modifying the repository during a run.** Commits, branch switches and merges in the repository while a worker finishes only cause pauses or extra revalidations.

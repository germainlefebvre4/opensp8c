package pool

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// MaxReviewBytes caps the patch and file content returned for a review.
const MaxReviewBytes = 512 * 1024

var (
	// ErrReviewBranchMissing reports that feature/<change> does not exist.
	ErrReviewBranchMissing = errors.New("la branche du changement est introuvable")
	// ErrInvalidReviewPath reports an empty, absolute or parent-escaping path.
	ErrInvalidReviewPath = errors.New("chemin de fichier invalide")
	// ErrReviewPathNotInReview reports a valid path the branch does not change.
	ErrReviewPathNotInReview = errors.New("fichier hors de la revue")
	// ErrReviewFileDeleted reports that the branch deletes the requested file,
	// so it has no final content.
	ErrReviewFileDeleted = errors.New("fichier supprimé par la branche")
)

// ReviewFileEntry is one file changed by the branch of a change.
type ReviewFileEntry struct {
	Path      string `json:"path"`
	Status    string `json:"status"` // added | modified | deleted
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Binary    bool   `json:"binary"`
}

// ReviewList is the set of files a change branch brings relative to the point
// where it diverged from Base.
type ReviewList struct {
	Base  string            `json:"base"`
	Files []ReviewFileEntry `json:"files"`
}

// ReviewContent is a patch or a file content, possibly truncated; Binary
// carries no content.
type ReviewContent struct {
	Content   string `json:"content"`
	Binary    bool   `json:"binary"`
	Truncated bool   `json:"truncated"`
}

// reviewBase resolves the base the review diffs against: the recorded base
// branch, else the current branch, else HEAD when that reference is gone.
func (wc *WorktreeController) reviewBase(changeName string) string {
	base, ok := wc.BaseBranch(changeName)
	if !ok {
		base = wc.CurrentBranch()
	}
	if base != "HEAD" {
		if _, code, err := wc.runGitCode("rev-parse", "--verify", "--quiet", "refs/heads/"+base); err != nil || code != 0 {
			return "HEAD"
		}
	}
	return base
}

// ReviewFiles lists the files feature/<change> changes relative to its point
// of divergence from the base (base...feature/<change>). It reads references
// only: the worktree is not needed. A rename shows as a deletion plus an
// addition.
func (wc *WorktreeController) ReviewFiles(changeName string) (ReviewList, error) {
	branch := "feature/" + changeName
	if exists, err := wc.branchExists(branch); err != nil {
		return ReviewList{}, err
	} else if !exists {
		return ReviewList{}, ErrReviewBranchMissing
	}
	base := wc.reviewBase(changeName)
	rng := base + "..." + branch

	numstat, err := wc.gitRaw("--literal-pathspecs", "diff", "--no-renames", "--numstat", "-z", rng, "--")
	if err != nil {
		return ReviewList{}, err
	}
	nameStatus, err := wc.gitRaw("--literal-pathspecs", "diff", "--no-renames", "--name-status", "-z", rng, "--")
	if err != nil {
		return ReviewList{}, err
	}

	byPath := map[string]*ReviewFileEntry{}
	for _, rec := range splitNUL(numstat) {
		parts := strings.SplitN(rec, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		entry := &ReviewFileEntry{Path: parts[2]}
		if parts[0] == "-" && parts[1] == "-" {
			entry.Binary = true
		} else {
			entry.Additions, _ = strconv.Atoi(parts[0])
			entry.Deletions, _ = strconv.Atoi(parts[1])
		}
		byPath[entry.Path] = entry
	}
	fields := splitNUL(nameStatus)
	for i := 0; i+1 < len(fields); i += 2 {
		entry, ok := byPath[fields[i+1]]
		if !ok {
			continue
		}
		switch fields[i] {
		case "A":
			entry.Status = "added"
		case "D":
			entry.Status = "deleted"
		default:
			entry.Status = "modified"
		}
	}

	files := make([]ReviewFileEntry, 0, len(byPath))
	for _, entry := range byPath {
		if entry.Status == "" {
			entry.Status = "modified"
		}
		files = append(files, *entry)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return ReviewList{Base: base, Files: files}, nil
}

// reviewEntry validates a client path then looks it up in the review list.
func (wc *WorktreeController) reviewEntry(changeName, path string) (ReviewList, ReviewFileEntry, error) {
	if !validReviewPath(path) {
		return ReviewList{}, ReviewFileEntry{}, ErrInvalidReviewPath
	}
	list, err := wc.ReviewFiles(changeName)
	if err != nil {
		return ReviewList{}, ReviewFileEntry{}, err
	}
	for _, f := range list.Files {
		if f.Path == path {
			return list, f, nil
		}
	}
	return ReviewList{}, ReviewFileEntry{}, ErrReviewPathNotInReview
}

// validReviewPath reports whether path is a plausible repository-relative file
// path: non-empty, relative, free of NUL and of ".." segments.
func validReviewPath(path string) bool {
	if path == "" || strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\x00\\") {
		return false
	}
	for _, seg := range strings.Split(path, "/") {
		if seg == ".." {
			return false
		}
	}
	return true
}

// ReviewPatch returns the unified patch of one file of the review, capped at
// MaxReviewBytes.
func (wc *WorktreeController) ReviewPatch(changeName, path string) (ReviewContent, error) {
	list, entry, err := wc.reviewEntry(changeName, path)
	if err != nil {
		return ReviewContent{}, err
	}
	if entry.Binary {
		return ReviewContent{Binary: true}, nil
	}
	out, truncated, err := wc.gitCapped(MaxReviewBytes,
		"--literal-pathspecs", "diff", "--no-renames", list.Base+"...feature/"+changeName, "--", path)
	if err != nil {
		return ReviewContent{}, err
	}
	return ReviewContent{Content: out, Truncated: truncated}, nil
}

// ReviewFile returns one file as it is in feature/<change>, capped at
// MaxReviewBytes. The size is read before the content so a huge blob is never
// loaded.
func (wc *WorktreeController) ReviewFile(changeName, path string) (ReviewContent, error) {
	_, entry, err := wc.reviewEntry(changeName, path)
	if err != nil {
		return ReviewContent{}, err
	}
	if entry.Status == "deleted" {
		return ReviewContent{}, ErrReviewFileDeleted
	}
	if entry.Binary {
		return ReviewContent{Binary: true}, nil
	}
	spec := "feature/" + changeName + ":" + path
	sizeOut, err := wc.gitRaw("cat-file", "-s", spec)
	if err != nil {
		return ReviewContent{}, err
	}
	size, _ := strconv.ParseInt(strings.TrimSpace(string(sizeOut)), 10, 64)
	out, truncated, err := wc.gitCapped(MaxReviewBytes, "show", spec)
	if err != nil {
		return ReviewContent{}, err
	}
	return ReviewContent{Content: out, Truncated: truncated || size > MaxReviewBytes}, nil
}

// splitNUL splits NUL-terminated git output into its records.
func splitNUL(b []byte) []string {
	var out []string
	for _, rec := range bytes.Split(b, []byte{0}) {
		if len(rec) > 0 {
			out = append(out, string(rec))
		}
	}
	return out
}

// gitRaw runs git in the repository and returns its stdout untouched (no
// trimming, unlike runGit, which would corrupt NUL-separated output).
func (wc *WorktreeController) gitRaw(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = wc.repoRoot
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %v failed: %w, stderr: %s", args, err, errOut.String())
	}
	return out.Bytes(), nil
}

// gitCapped runs git and reads at most limit bytes of stdout; when more was
// available, the process is stopped and truncated is true.
func (wc *WorktreeController) gitCapped(limit int, args ...string) (string, bool, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = wc.repoRoot
	var errOut bytes.Buffer
	cmd.Stderr = &errOut
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", false, err
	}
	if err := cmd.Start(); err != nil {
		return "", false, fmt.Errorf("git %v failed: %w", args, err)
	}
	buf, readErr := io.ReadAll(io.LimitReader(stdout, int64(limit)+1))
	truncated := len(buf) > limit
	if truncated {
		buf = buf[:limit]
		_ = cmd.Process.Kill()
	} else if readErr == nil {
		// Drain so a process blocked on a full pipe can finish.
		_, _ = io.Copy(io.Discard, stdout)
	}
	waitErr := cmd.Wait()
	if readErr != nil {
		return "", false, readErr
	}
	if waitErr != nil && !truncated {
		return "", false, fmt.Errorf("git %v failed: %w, stderr: %s", args, waitErr, errOut.String())
	}
	return strings.ToValidUTF8(string(buf), "�"), truncated, nil
}

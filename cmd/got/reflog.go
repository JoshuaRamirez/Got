package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joshuaramirez/got/internal/history"
	"github.com/joshuaramirez/got/internal/repo"
)

// A reflogEntry records one movement of a ref (a branch's commit tip, or the
// special ref "HEAD"). Unlike the commit DAG, which is content-addressed and
// only grows, the reflog is an ordered, append-only journal of where refs
// pointed over time — so a commit dropped by reset, rebase, or amend is still
// reachable for recovery. This is Got's analogue of git's reflog.
type reflogEntry struct {
	Ref     string `json:"ref"`     // "HEAD" or a branch name
	Old     string `json:"old"`     // previous commit hex ("" if the ref was unborn)
	New     string `json:"new"`     // new commit hex
	Action  string `json:"action"`  // commit, checkout, reset, merge, rebase, amend, revert, cherry-pick, branch
	Message string `json:"message"` // human context
	Time    string `json:"time"`    // RFC3339 UTC
}

func reflogPath() string { return filepath.Join(stateDir(), "reflog.json") }

func loadReflog() ([]reflogEntry, error) {
	b, err := os.ReadFile(reflogPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []reflogEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, fmt.Errorf("corrupt reflog file: %w", err)
	}
	return entries, nil
}

// appendReflog adds one entry to the journal. Reflog writes are best-effort:
// callers ignore the error so a journaling failure never aborts the operation
// that actually moved the ref.
func appendReflog(e reflogEntry) error {
	entries, err := loadReflog()
	if err != nil {
		return err
	}
	entries = append(entries, e)
	if err := os.MkdirAll(stateDir(), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := reflogPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, reflogPath())
}

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

func commitHex(id history.CommitID) string { return hex.EncodeToString(id[:]) }

// setBranchTip binds a branch's commit ref to newID and journals the move to
// the reflog. When the branch is the current branch, a mirror "HEAD" entry is
// written too, so `got reflog` (which defaults to HEAD) shows the full activity
// stream just like git. The old tip is resolved before the bind.
func setBranchTip(state repo.State, branch string, newID history.CommitID, action, message string) error {
	ctx := context.Background()
	old := ""
	if id, ok := state.Namespace().ResolveRef(ctx, commitRefName(branch)); ok {
		old = commitHex(commitFromVID(id))
	}
	if err := state.Namespace().BindRef(ctx, commitRefName(branch), vidFromCommit(newID)); err != nil {
		return err
	}
	at := nowUTC()
	_ = appendReflog(reflogEntry{Ref: branch, Old: old, New: commitHex(newID), Action: action, Message: message, Time: at})
	if branch == currentBranch() {
		_ = appendReflog(reflogEntry{Ref: "HEAD", Old: old, New: commitHex(newID), Action: action, Message: message, Time: at})
	}
	return nil
}

// logHEADMove journals a HEAD movement that does not itself change a branch tip
// (checkout/switch between branches). old/new are the tips of the branches HEAD
// moved between, so the journal stays commit-addressed like git's HEAD reflog.
func logHEADMove(oldTip, newTip, message string) {
	_ = appendReflog(reflogEntry{Ref: "HEAD", Old: oldTip, New: newTip, Action: "checkout", Message: message, Time: nowUTC()})
}

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/joshuaramirez/got/internal/history"
	"github.com/joshuaramirez/got/internal/repo"
)

// bisectState is the persisted session for a binary search over history. Good
// and Bad are the current boundary (both narrow as the search proceeds); Current
// is the candidate whose snapshot is checked out in the working graph awaiting a
// verdict; OrigBranch is restored on `bisect reset`.
type bisectState struct {
	OrigBranch string `json:"orig_branch"`
	Good       string `json:"good"`
	Bad        string `json:"bad"`
	Current    string `json:"current"`
}

func bisectPath() string { return filepath.Join(stateDir(), "bisect.json") }

func loadBisect() (*bisectState, bool, error) {
	b, err := os.ReadFile(bisectPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var s bisectState
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, false, fmt.Errorf("corrupt bisect file: %w", err)
	}
	return &s, true, nil
}

func saveBisect(s *bisectState) error {
	if err := os.MkdirAll(stateDir(), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := bisectPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, bisectPath())
}

func clearBisect() error {
	err := os.Remove(bisectPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// cmdBisect binary-searches the commit range (good, bad] for the first commit at
// which a predicate flips from good to bad — the graph analogue of git bisect.
// Unlike git it does not move HEAD; it restores each candidate's snapshot to the
// working graph so the predicate (or the developer) inspects that state.
func cmdBisect(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "bisect: expected start|good|bad|run|reset|status")
		return 2
	}
	switch args[0] {
	case "start":
		return bisectStart(args[1:], stdout, stderr)
	case "good":
		return bisectVerdict(false, stdout, stderr)
	case "bad":
		return bisectVerdict(true, stdout, stderr)
	case "run":
		return bisectRun(args[1:], stdout, stderr)
	case "reset":
		return bisectReset(stdout, stderr)
	case "status":
		return bisectStatus(stdout, stderr)
	default:
		fmt.Fprintf(stderr, "bisect: unknown subcommand %q\n", args[0])
		return 2
	}
}

func bisectStart(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "bisect: expected start <bad> <good>")
		return 2
	}
	state, log, ok := loadStateLog(stderr, "bisect")
	if !ok {
		return 1
	}
	bad, ok := resolveCommit(state, log, args[0])
	if !ok {
		fmt.Fprintf(stderr, "bisect: unknown bad commit %q\n", args[0])
		return 1
	}
	good, ok := resolveCommit(state, log, args[1])
	if !ok {
		fmt.Fprintf(stderr, "bisect: unknown good commit %q\n", args[1])
		return 1
	}
	if bad == good {
		fmt.Fprintln(stderr, "bisect: bad and good are the same commit")
		return 1
	}
	badAnc, err := ancestorSet(log, bad)
	if err != nil {
		fmt.Fprintf(stderr, "bisect: %v\n", err)
		return 1
	}
	if !badAnc[good] {
		fmt.Fprintln(stderr, "bisect: good is not an ancestor of bad")
		return 1
	}

	s := &bisectState{OrigBranch: currentBranch(), Good: commitHex(good), Bad: commitHex(bad)}
	return bisectAdvance(s, state, log, stdout, stderr)
}

func bisectVerdict(isBad bool, stdout, stderr io.Writer) int {
	s, running, err := loadBisect()
	if err != nil {
		fmt.Fprintf(stderr, "bisect: %v\n", err)
		return 1
	}
	if !running {
		fmt.Fprintln(stderr, "bisect: no bisect in progress (run 'got bisect start')")
		return 1
	}
	state, log, ok := loadStateLog(stderr, "bisect")
	if !ok {
		return 1
	}
	if s.Current == "" {
		fmt.Fprintln(stderr, "bisect: no candidate to mark")
		return 1
	}
	if isBad {
		s.Bad = s.Current
	} else {
		s.Good = s.Current
	}
	return bisectAdvance(s, state, log, stdout, stderr)
}

// bisectAdvance recomputes the suspect set for the current (good, bad] boundary
// and either checks out the next candidate to test or, when the range is empty,
// concludes that Bad is the first bad commit.
func bisectAdvance(s *bisectState, state repo.State, log *history.Log, stdout, stderr io.Writer) int {
	bad, err := decodeCommitHex(s.Bad)
	if err != nil {
		fmt.Fprintf(stderr, "bisect: %v\n", err)
		return 1
	}
	good, err := decodeCommitHex(s.Good)
	if err != nil {
		fmt.Fprintf(stderr, "bisect: %v\n", err)
		return 1
	}
	suspects, err := suspectSet(log, bad, good)
	if err != nil {
		fmt.Fprintf(stderr, "bisect: %v\n", err)
		return 1
	}
	if len(suspects) == 0 {
		// Nothing left strictly between good and bad: bad is the first bad commit.
		s.Current = ""
		if err := saveBisect(s); err != nil {
			fmt.Fprintf(stderr, "bisect: %v\n", err)
			return 1
		}
		c, _ := log.Get(bad)
		fmt.Fprintf(stdout, "%s is the first bad commit: %s\n", shortID(bad[:]), c.Message)
		fmt.Fprintln(stdout, "(run 'got bisect reset' to end the session)")
		return 0
	}
	next := pickCandidate(log, suspects)
	s.Current = commitHex(next)
	if err := saveBisect(s); err != nil {
		fmt.Fprintf(stderr, "bisect: %v\n", err)
		return 1
	}
	if err := restoreWorking(state, log, next); err != nil {
		fmt.Fprintf(stderr, "bisect: %v\n", err)
		return 1
	}
	c, _ := log.Get(next)
	fmt.Fprintf(stdout, "bisecting: %d commit(s) left to test\n", len(suspects))
	fmt.Fprintf(stdout, "  now testing %s: %s\n", shortID(next[:]), c.Message)
	return 0
}

func bisectRun(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "bisect: expected run <command> [args...]")
		return 2
	}
	s, running, err := loadBisect()
	if err != nil {
		fmt.Fprintf(stderr, "bisect: %v\n", err)
		return 1
	}
	if !running {
		fmt.Fprintln(stderr, "bisect: no bisect in progress (run 'got bisect start')")
		return 1
	}
	// Iterate until the session concludes (Current cleared) or an error occurs.
	for {
		s, running, err = loadBisect()
		if err != nil {
			fmt.Fprintf(stderr, "bisect: %v\n", err)
			return 1
		}
		if !running || s.Current == "" {
			return 0
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		runErr := cmd.Run()
		isBad := runErr != nil
		verdict := "good"
		if isBad {
			verdict = "bad"
		}
		fmt.Fprintf(stdout, "  predicate says %s\n", verdict)
		if code := bisectVerdict(isBad, stdout, stderr); code != 0 {
			return code
		}
	}
}

func bisectReset(stdout, stderr io.Writer) int {
	s, running, err := loadBisect()
	if err != nil {
		fmt.Fprintf(stderr, "bisect: %v\n", err)
		return 1
	}
	if !running {
		fmt.Fprintln(stderr, "bisect: no bisect in progress")
		return 1
	}
	state, log, ok := loadStateLog(stderr, "bisect")
	if !ok {
		return 1
	}
	// Restore the working graph to the branch we started on.
	if id, ok := state.Namespace().ResolveRef(context.Background(), commitRefName(s.OrigBranch)); ok {
		if err := restoreWorking(state, log, commitFromVID(id)); err != nil {
			fmt.Fprintf(stderr, "bisect: %v\n", err)
			return 1
		}
	}
	if err := clearBisect(); err != nil {
		fmt.Fprintf(stderr, "bisect: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "bisect reset: back on %s\n", s.OrigBranch)
	return 0
}

func bisectStatus(stdout, stderr io.Writer) int {
	s, running, err := loadBisect()
	if err != nil {
		fmt.Fprintf(stderr, "bisect: %v\n", err)
		return 1
	}
	if !running {
		fmt.Fprintln(stdout, "no bisect in progress")
		return 0
	}
	fmt.Fprintf(stdout, "bisect in progress (started on %s)\n", s.OrigBranch)
	fmt.Fprintf(stdout, "  bad:  %s\n", short(s.Bad))
	fmt.Fprintf(stdout, "  good: %s\n", short(s.Good))
	if s.Current != "" {
		fmt.Fprintf(stdout, "  testing: %s\n", short(s.Current))
	}
	return 0
}

// --- helpers ---

// loadStateLog loads both the repository state and commit log, reporting under
// the given command name on failure.
func loadStateLog(stderr io.Writer, cmd string) (repo.State, *history.Log, bool) {
	state, err := loadState()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return nil, nil, false
	}
	log, err := loadHistory()
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", cmd, err)
		return nil, nil, false
	}
	return state, log, true
}

func short(hexID string) string {
	if len(hexID) >= 8 {
		return hexID[:8]
	}
	return hexID
}

// restoreWorking sets the working graph to a commit's snapshot (bisect is
// detached: HEAD/branch pointers are left untouched).
func restoreWorking(state repo.State, log *history.Log, id history.CommitID) error {
	c, ok := log.Get(id)
	if !ok {
		return fmt.Errorf("unknown commit %s", hex.EncodeToString(id[:]))
	}
	g, err := c.Snapshot.Build(schema())
	if err != nil {
		return err
	}
	return saveState(repo.NewState(g, state.Namespace()))
}

func ancestorSet(log *history.Log, id history.CommitID) (map[history.CommitID]bool, error) {
	anc, err := log.Ancestors(id)
	if err != nil {
		return nil, err
	}
	set := make(map[history.CommitID]bool, len(anc))
	for _, c := range anc {
		set[c.ID] = true
	}
	return set, nil
}

// suspectSet returns the commits strictly between good and bad: ancestors of bad
// that are not ancestors of good, excluding bad itself. The first bad commit is
// bad if this set is empty, otherwise somewhere in it.
func suspectSet(log *history.Log, bad, good history.CommitID) ([]history.CommitID, error) {
	badAnc, err := ancestorSet(log, bad)
	if err != nil {
		return nil, err
	}
	goodAnc, err := ancestorSet(log, good)
	if err != nil {
		return nil, err
	}
	var out []history.CommitID
	for id := range badAnc {
		if id == bad {
			continue
		}
		if goodAnc[id] {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

// pickCandidate chooses the optimal bisection point: the suspect whose ancestor
// count within the suspect set most evenly splits it, so either verdict discards
// roughly half. Ties break on the lexicographically smallest id for determinism.
func pickCandidate(log *history.Log, suspects []history.CommitID) history.CommitID {
	inSet := make(map[history.CommitID]bool, len(suspects))
	for _, id := range suspects {
		inSet[id] = true
	}
	n := len(suspects)
	best := suspects[0]
	bestScore := -1
	bestKey := ""
	for _, c := range suspects {
		anc, err := log.Ancestors(c)
		if err != nil {
			continue
		}
		a := 0
		for _, ac := range anc {
			if inSet[ac.ID] {
				a++
			}
		}
		score := a
		if n-a < score {
			score = n - a
		}
		key := hex.EncodeToString(c[:])
		if score > bestScore || (score == bestScore && key < bestKey) {
			bestScore = score
			best = c
			bestKey = key
		}
	}
	return best
}

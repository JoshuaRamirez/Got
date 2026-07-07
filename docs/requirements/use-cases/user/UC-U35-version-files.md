# UC-U35: Version real files through the graph

| Field | Value |
|---|---|
| Goal level | User goal (sea) |
| Scope | `cmd/got` (`add`, `extract`), `internal/history` (content-addressed commit id) |
| Primary actor | Developer |
| Stakeholders & interests | Developer: put actual source files under version control — ingest them, commit, branch, merge, and check a chosen commit's files back out to disk — not just abstract graph nodes. |
| Preconditions | The repository is initialized. |
| Trigger | `got add <path>...` then the normal commit/branch/merge flow, and `got extract [<dir>]`. |
| Success postcondition | Each committed state's file bytes (and permission bits) are recoverable exactly; switching branches and extracting renders that branch's files. |
| Failure postcondition | An error is returned; nothing is written. |

## Main success scenario

1. Developer runs `got add <path>...`. System reads each file (walking
   directories, skipping the state dir and `.git`) and upserts it into the
   working graph as an `Artifact` vertex named by its repo-relative path, with
   the bytes base64-encoded under `file.content` and permission bits under
   `file.mode`.
2. Developer runs `got commit` — the working graph, file vertices included, is
   snapshotted into a commit (UC-U22). The commit id folds in each element's
   content digest, so an edit at the same path yields a distinct commit even
   with an identical message.
3. Developer branches, edits files on disk, `add`s and commits again; merges,
   rebases, reverts, etc. all operate on the file vertices for free because they
   are ordinary graph content.
4. Developer runs `got extract [<dir>]`. System writes every file vertex in the
   working graph to `<dir>` (default `.`), recreating directories and restoring
   permission bits.

## Extensions

### Successful variations

- **1a. Directory ingest:** a directory argument is walked recursively; regular
  files are added, VCS metadata is skipped.
- **1b. Re-add after edit:** adding a path that already has a vertex replaces its
  content (the vertex id is stable on the path).
- **3a. Disjoint merge:** two branches editing different files merge cleanly and
  both edits survive — the per-vertex merge needs no line-based patching.

### Failure paths

- **4a. Path traversal:** a file vertex whose path is absolute or escapes the
  target directory is refused on `extract` (`safeJoin`).
- **1c. Missing path:** `add` of a nonexistent path reports the OS error and
  fails.
- **0a. No repository:** `add`/`extract` before `got init` report the
  `run 'got init'` hint.

## Sub-variations

- **Content-addressed identity:** because a file vertex's id is
  `sha256(path)`, two trees differing only in a file's bytes share vertex ids;
  the commit id therefore hashes each element's full content digest, not its
  bare id, so in-place edits are distinct commits (`history.computeID`).

## Known limitations (honest scope)

- **Per-file, not per-line, merge:** a file's whole content is one vertex
  attribute, so two branches editing the *same* file conflict at the file level
  (resolvable with `merge --ours`/`--theirs`, UC-U32) — coarser than git's
  hunk-level merge.
- **No blob dedup:** content is stored inline in each snapshot; identical files
  at different paths (or unchanged files across commits) are not yet
  deduplicated into a shared content-addressed blob store.

## Related use cases

- Uses: UC-U22 (commit), UC-U23 (checkout — rebuilds the working graph that
  `extract` renders), UC-S23 (snapshot codec). Interacts with UC-U24/UC-U32
  (merge and merge strategies) for file-level reconciliation.

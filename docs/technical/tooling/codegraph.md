# CodeGraph

Indexes the repo for structural exploration (symbols, callers/callees, change impact) without text-search loops.

Use for architecture and blast-radius questions; use grep/read for literal strings.

Re-index when `.codegraph/` is missing, after large refactors, or when results look stale:

```bash
codegraph init -i
```

Daily small edits usually do not need a full re-index (file watcher keeps the index warm).

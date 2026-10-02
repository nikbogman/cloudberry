## Working in this repo

- Check: `go vet ./... && go test ./...`
- Deploy: `task --dry` first, then `task` or one target (`task --list`). Manual only; run it when the user asks. Secrets and addresses come from the root `.env` (template `.env.example`); see `DEPLOY.md`.
- edge deploys through Railway, never through `task`; see `cmd/edge/README.md#deployment`.
- Stacks deploy with `docker compose --context blackberry`, never through `task`; see `stacks/README.md`.
- New Go service: `cmd/<name>/` + `internal/<name>/`, plus a task in `Taskfile.yml` that calls `_service`.

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).

## Domain docs

- Vocabulary: `CONTEXT.md` (single context). Name domain concepts by its terms, never the synonyms it lists under _Avoid_. A missing term is either invented language or a real gap; raise gaps for `/domain-modeling`.
- Decisions: `DESIGN.md#design-decisions`, one bullet each with its reason. Record new ones there, not in `docs/adr/`. When your output contradicts one, say so explicitly.

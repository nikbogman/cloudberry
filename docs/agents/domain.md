# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- **`CONTEXT.md`** at the repo root — this repo is single-context, so there's exactly one.
- **`docs/adr/DECISIONS.md`** — decisions touching the area you're about to work in. A deliberate deviation from the default one-`NNNN-slug.md`-per-decision convention: related decisions from the same design session are kept together as sections in one file, and new ones are appended there rather than as separate numbered files.

`DECISIONS.md` isn't created upfront and doesn't exist yet. If it's missing, **proceed silently** — don't flag its absence, don't suggest creating it. The `/domain-modeling` skill (reached via `/grill-with-docs` and `/improve-codebase-architecture`) creates it lazily when decisions actually get resolved.

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a test name), use the term as defined in `CONTEXT.md`. Don't drift to synonyms the glossary explicitly avoids.

If the concept you need isn't in the glossary yet, that's a signal — either you're inventing language the project doesn't use (reconsider) or there's a real gap (note it for `/domain-modeling`).

## Flag ADR conflicts

If your output contradicts an existing decision in `docs/adr/DECISIONS.md`, surface it explicitly rather than silently overriding:

> _Contradicts the "Hold only blocks automatic suspend" decision in DECISIONS.md — but worth reopening because…_

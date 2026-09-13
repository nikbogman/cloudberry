# Scope the AI agent workload

Status: needs-info

## Problem

The AI agent workload is named in the spec but explicitly not scoped (spec Out of Scope, User Story 11). Its compose file can't be written against guesses.

## Open Questions

- Which agent/software is it?
- GPU passthrough required?
- Model storage path/size under `/srv/stacks/<name>/`?

## Notes

Follow-up only — not part of this spec's build. Comes after issues 02/03 prove the pattern (spec: "First implementation order"). Once answered, this becomes a normal `stacks/<name>/` ticket following the same shape as issue 02.

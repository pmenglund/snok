# AGENTS.md

Instructions for Codex and other coding agents working in this repository.

## Working Safely

- Check `git status --short` before editing. The user may make changes between
  turns; preserve unrelated local modifications and do not revert user work
  unless explicitly asked.
- Keep changes scoped to the requested behavior. Avoid broad refactors when a
  focused patch will solve the task.
- Prefer `rg` and `rg --files` for repo inspection.
- Use `apply_patch` for manual file edits.

## Repository Contract

- Treat `SPEC.md` as the high-level behavior contract for Snok.
- Treat `specs/fixtures/*.json`, `specs/conformance.md`, and `specs/evidence.md`
  as conformance and evidence material. Do not change them casually; update them
  only when the task explicitly changes the behavior contract or fixture set.
- Keep README examples aligned with checked-in code under `examples/` and
  `cmd/snok`.
- Keep docs and examples accurate for the current public API names, including
  `NewTree`, `AddCommand`, `AddGroup`, `CommandDefinition`, `GroupDefinition`,
  `Field`, `RunCommand`, and `CobraCommand`.
- When adding or restructuring Snok-powered app commands, consult
  `docs/codex-command-authoring.md` for the expected command registration,
  structure, output, error, and testing patterns.

## Go Development

- Snok is a Go module at `github.com/pmenglund/snok`.
- Use idiomatic Go and preserve the existing small-package layout unless the
  requested change requires moving code.
- Run focused tests for touched packages, then run `go test ./...` for broader
  changes.
- Use `gofmt` on modified Go files.
- Do not add new dependencies without a clear need and without checking the
  existing standard-library or local-code option first.

## Documentation

- Keep `README.md` human-facing. It should explain what Snok is, how to install
  and use it, runnable examples, CLI workflows, and development commands for
  maintainers and users.
- Keep `AGENTS.md` agent-facing. It should contain repository operating
  instructions for Codex and other coding agents: safety rules, source-of-truth
  guidance, testing expectations, and documentation maintenance boundaries.
- Do not duplicate the README's user tutorial content in `AGENTS.md`; point
  agents at the README when they need user-facing usage context.
- Keep human-facing docs practical and runnable. Prefer commands and code that
  match the repository's current examples.
- When documenting behavior covered by the spec, link or point to `SPEC.md`
  instead of duplicating every normative detail.
- If docs describe CLI commands, verify the command names and arguments against
  `cmd/snok/main.go`.

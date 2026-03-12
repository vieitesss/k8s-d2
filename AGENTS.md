# AGENTS.md

This repo is a Go CLI named `k8sdd` that inspects Kubernetes clusters and renders D2 topology diagrams.

## Requirements

- Keep the work aligned with the current feature or target scope.
- Use `PLAN.md` in the repo root as the living plan for the current feature or update.
- `PLAN.md` must stay ignored by git through `.gitignore`.
- Create or refresh `PLAN.md` before starting work, and update it after every implementation, fix, or scope change.
- Never push directly to `main`.
- If the current branch is `main`, create a new branch with a name that summarizes the goal before making changes, then create a detailed `PLAN.md` for that work.
- Before finishing, run the relevant tests. If new behavior was added and no test covers it yet, add the missing coverage.
- Pay special attention to validation-related coverage in `internal/validation/`.

## Implementation Notes

- 2026-03-12: Rewrote this file to center the workflow on planning, branch safety, and test verification. The old content described project phases and stack details but did not capture the required `PLAN.md` process.
- Keep adding short notes here when something was wrong, how it was fixed, or what future implementations should remember.

## Current Plan

- Active implementation details live in `PLAN.md`.

## References

- Coding patterns: `docs/GO_PATTERNS.md`
- Testing guidance: `docs/TESTING.md`
- Git workflow: `docs/GIT_WORKFLOW.md`
- Development commands: `docs/DEVELOPMENT.md`

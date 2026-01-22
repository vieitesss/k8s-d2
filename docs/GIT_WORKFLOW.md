# Git Workflow

Commit standards and PR review procedures for k8sdd.

## Commit Standards

Use conventional commits:

- `feat:` - New feature
- `fix:` - Bug fix
- `refactor:` - Code restructuring without behavior change
- `docs:` - Documentation updates
- `chore:` - Maintenance tasks

Keep commits focused on single concern.

Include issue reference if applicable: `feat: add ingress support (#12)`

## PR Review Checklist

When reviewing pull requests, verify:

### Code Quality

- [ ] No new global variables (use RootOptions pattern)
- [ ] Error handling follows spinner wrapper pattern
- [ ] D2 IDs are sanitized (no hyphens)
- [ ] Functions have cyclomatic complexity ≤ 15
- [ ] Code is formatted (`go fmt`)

### Architecture

- [ ] Changes follow three-layer architecture
- [ ] K8s API calls use `metav1.ListOptions{}`
- [ ] Context passed from cobra command to client methods
- [ ] New flags added to RootOptions struct

### Testing

- [ ] Binary builds successfully
- [ ] Output produces valid D2 syntax
- [ ] Changes work with real K8s cluster
- [ ] Edge cases handled (empty namespaces, no selectors, etc.)

### Documentation

- [ ] README updated if adding user-facing feature
- [ ] CLAUDE.md updated if changing architecture or design decisions
- [ ] docs/ updated if changing coding patterns
- [ ] Code comments explain "why" not "what"

## Release Process

See `.goreleaser.yml` for release automation configuration.

# AGENTS.md

> **File Purpose**: Machine-readable instructions for AI coding agents (GitHub Copilot, Claude Code). Defines coding standards, workflows, and actionable guidelines.
>
> **Relationship to CLAUDE.md**: This file focuses on HOW to code (patterns, standards, checklists). For WHY architectural decisions were made and WHAT the system design is, see CLAUDE.md.

## Project Identity

**What**: CLI tool generating D2 diagrams from Kubernetes cluster topology
**Language**: Go 1.24.0
**Architecture**: Three-layer (CLI → Data → Render)
**Binary Name**: `k8sdd`
**Entry Point**: `main.go`

## Tech Stack

### Core Dependencies
- `github.com/spf13/cobra` - CLI framework
- `k8s.io/client-go` v0.29.0 - Kubernetes API client
- `github.com/charmbracelet/log` - Structured logging
- `github.com/charmbracelet/huh/spinner` - Terminal UI spinners

### Standard Libraries
- `context.Context` - Always pass from cobra commands to client methods
- `io.Writer` - Used for rendering output (file or stdout)
- `strings.Builder` - Prefer for D2 syntax construction

## Commands

### Build and Test
```bash
# Build binary
go build -o k8sdd .

# Run without building
go run main.go -o cluster.d2

# Test specific namespace
go run main.go -n <namespace> -o test.d2

# Verify build after changes
go build -o k8sdd . && ./k8sdd -o /tmp/test.d2

# Run against live cluster (requires valid kubeconfig)
./k8sdd --all-namespaces -o output.d2
```

### Code Quality
```bash
# Format code (always run before committing)
go fmt ./...

# Run linter
golangci-lint run

# Check cyclomatic complexity
gocyclo -over 15 .
```

## Project Structure

```
cmd/
  root.go       # Cobra command setup, flag definitions, RootOptions struct
  generate.go   # Main command logic: spinner wrappers, orchestration flow
pkg/
  kube/
    client.go   # K8s clientset initialization, klog suppression
    fetch.go    # Resource fetching, namespace filtering, namespaceFetcher pattern
  model/
    types.go    # Internal data model: Cluster/Namespace/Workload/Service/PVC
  render/
    d2.go       # D2 syntax generation, ID sanitization, label matching
main.go         # Application entry point, version injection
```

### Where to Add Code

| Task | Location | Example Reference |
|------|----------|-------------------|
| New CLI flag | `cmd/root.go` RootOptions struct | `rootOptions.gridColumns` |
| New K8s resource fetch | `pkg/kube/fetch.go` | `fetchDeployments`, `fetchServices` |
| New data model field | `pkg/model/types.go` | `Workload.Labels`, `Service.Ports` |
| D2 rendering logic | `pkg/render/d2.go` | `writeWorkload`, `writeService` |
| Error handling | Function that creates client/fetches | `createClientWithSpinner` |

## Code Style

### DO

**Error Handling**
```go
// Wrap operations with spinner + separate error handling
var result *Type
var err error
spinnerErr := spinner.New().
    Title("Doing operation...").
    Action(func() {
        result, err = doOperation()
    }).
    Run()
if spinnerErr != nil {
    return nil, spinnerErr
}
return result, err
```

**ID Sanitization** (D2 requires safe identifiers)
```go
// Always sanitize before using as D2 ID
sanitizedID := sanitizeID(resourceName) // "my-service" -> "my_service"
```

**Label Matching** (Service selectors → Workload labels)
```go
// See labelsMatch() in pkg/render/d2.go
// Selector is subset of labels, all keys must match
```

**Namespace Fetchers** (Type alias pattern)
```go
type namespaceFetcher func(context.Context, string, *model.Namespace) error

fetchers := []namespaceFetcher{
    c.fetchDeployments,
    c.fetchStatefulSets,
}
```

### DON'T

- Don't add global variables - use `RootOptions` struct pattern
- Don't write to stderr during spinner operations (breaks UI)
- Don't use hyphens in D2 IDs (use `sanitizeID()`)
- Don't add flags without updating `RootOptions` in `cmd/root.go`
- Don't suppress klog output anywhere except `pkg/kube/client.go` init()

### Code Patterns from Real Files

**Best examples to follow:**
- `cmd/root.go`: RootOptions struct pattern for flags
- `pkg/kube/fetch.go`: namespaceFetcher type alias, sequential fetching
- `pkg/render/d2.go`: strings.Builder for text generation, helper methods
- `cmd/generate.go`: Spinner wrapper pattern, error propagation

**Legacy patterns to avoid:**
- Individual package-level flag variables (use RootOptions)
- Direct stderr writing (use charmbracelet/log)

## Testing

### Manual Testing Flow
1. Build: `go build -o k8sdd .`
2. Run against test namespace: `./k8sdd -n <test-ns> -o test.d2`
3. Verify D2 syntax: `d2 test.d2 test.svg`
4. Check output has: namespaces, workloads with icons, services, connections

### What to Verify

**After adding K8s resource support:**
- Resource appears in model (`pkg/model/types.go`)
- Fetch function exists (`pkg/kube/fetch.go`)
- Renderer handles it (`pkg/render/d2.go`)
- D2 output is valid (test with `d2` CLI)

**After adding CLI flag:**
- Flag defined in `RootOptions` struct
- Flag bound in `init()` function
- Flag accessed via `rootOptions.fieldName`
- Help text accurate (`k8sdd --help`)

## Git Workflow

### Commit Standards
- Use conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`
- Keep commits focused on single concern
- Include issue reference if applicable: `feat: add ingress support (#12)`

### PR Review Checklist
When reviewing PRs, check:

**Code Quality**
- [ ] No new global variables (use RootOptions pattern)
- [ ] Error handling follows spinner wrapper pattern
- [ ] D2 IDs are sanitized (no hyphens)
- [ ] Functions have cyclomatic complexity ≤ 15
- [ ] Code is formatted (`go fmt`)

**Architecture**
- [ ] Changes follow three-layer architecture
- [ ] K8s API calls use `metav1.ListOptions{}`
- [ ] Context passed from cobra command to client methods
- [ ] New flags added to RootOptions struct

**Testing**
- [ ] Binary builds successfully
- [ ] Output produces valid D2 syntax
- [ ] Changes work with real K8s cluster
- [ ] Edge cases handled (empty namespaces, no selectors, etc.)

**Documentation**
- [ ] README updated if adding user-facing feature
- [ ] CLAUDE.md updated if changing architecture or design decisions
- [ ] AGENTS.md updated if changing coding patterns or AI workflows
- [ ] Code comments explain "why" not "what"

## Boundaries

### ✅ ALLOWED WITHOUT ASKING

- Add new K8s resource fetchers following existing patterns
- Extend model types with new fields
- Add D2 rendering for new resource types
- Fix bugs in error handling or output
- Improve code quality (reduce complexity, better names)
- Add CLI flags following RootOptions pattern
- Update dependencies (with justification)

### ❌ REQUIRES DISCUSSION

- Change three-layer architecture
- Remove existing CLI flags (breaking change)
- Change D2 output format significantly (breaks user workflows)
- Add external service dependencies
- Change license or add CLA
- Modify build/release process (.goreleaser.yml)

### 🔴 NEVER

- Break client-go v0.29.0 compatibility
- Remove namespace filtering logic
- Add telemetry or analytics without explicit opt-in
- Commit binaries or generated files to git
- Suppress errors that should be shown to user

## Implementation Phases

Current development follows phased approach:

- **Phase 1** ✅ Complete: Basic topology (namespaces, workloads, services, configmaps/secrets)
- **Phase 2** 🚧 In Progress: Storage layer (PVCs, StorageClasses, volume relationships)
- **Phase 3** 📋 Planned: Network layer (Ingress, NetworkPolicies, pod connections)

When implementing features:
- Respect phase boundaries
- Don't jump ahead to phase 3 features
- Focus on completing current phase fully

## Special Contexts

### For GitHub Copilot (PR Reviews)

When reviewing pull requests, pay special attention to:

1. **RootOptions Pattern**: All new flags must be struct fields, not globals
2. **Cyclomatic Complexity**: Functions should stay under complexity 15
3. **Error Handling**: Both spinner errors AND operation errors must be checked
4. **D2 Syntax**: Output must be valid (IDs sanitized, proper nesting)
5. **Breaking Changes**: Flag removals or output format changes need discussion

### For Claude Code

When generating code:

1. **Always read existing code first** before suggesting changes
2. **Follow established patterns** from referenced files
3. **Run `go build`** after significant changes
4. **Test with real cluster** if possible (or mock with clear TODO)
5. **Keep cyclomatic complexity low** - break into helpers if >15

## Quick Reference

**Most frequently modified files:**
- `cmd/root.go` - Adding CLI flags
- `pkg/kube/fetch.go` - Adding K8s resource fetchers
- `pkg/render/d2.go` - Changing D2 output

**Most frequently used patterns:**
- RootOptions struct for flags
- namespaceFetcher type alias for resource fetchers
- Spinner wrapper with dual error checking
- sanitizeID for D2 compatibility
- strings.Builder for text generation

**Common tasks:**
1. Add new K8s resource: model → fetch → render
2. Add CLI flag: RootOptions → init() → use in code
3. Fix complexity: extract helper methods, reduce nesting
4. Debug output: `go run main.go -n test -o /tmp/out.d2 && cat /tmp/out.d2`

---

*This file is version-controlled and should be updated when architectural patterns change. Last updated: 2025-12-22*

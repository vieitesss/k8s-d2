# AGENTS.md

CLI tool generating D2 diagrams from Kubernetes cluster topology.

**Language**: Go 1.24.0  
**Architecture**: Three-layer (CLI → Data → Render)  
**Binary**: `k8sdd`

## Core Stack

- `github.com/spf13/cobra` - CLI framework
- `k8s.io/client-go` v0.29.0 - Kubernetes API client
- `github.com/charmbracelet/log` - Structured logging
- `github.com/charmbracelet/huh/spinner` - Terminal UI spinners

## Implementation Phases

Current development follows phased approach:

- **Phase 1** ✅ Complete: Basic topology (namespaces, workloads, services, configmaps/secrets)
- **Phase 2** 🚧 In Progress: Storage layer (PVCs, StorageClasses, volume relationships)
- **Phase 3** 📋 Planned: Network layer (Ingress, NetworkPolicies, pod connections)

When implementing features:
- Respect phase boundaries
- Don't jump ahead to phase 3 features
- Focus on completing current phase fully

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

## Domain-Specific Guidance

For coding patterns and style, see [docs/GO_PATTERNS.md](docs/GO_PATTERNS.md)  
For testing procedures, see [docs/TESTING.md](docs/TESTING.md)  
For Git workflow and PR reviews, see [docs/GIT_WORKFLOW.md](docs/GIT_WORKFLOW.md)  
For development commands, see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)

For architectural decisions and system design, see [CLAUDE.md](CLAUDE.md)

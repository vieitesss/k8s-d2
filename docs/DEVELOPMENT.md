# Development Commands

Common development tasks and commands for k8sdd.

## Build and Run

```bash
# Build binary
go build -o k8sdd .

# Run without building
go run main.go -o cluster.d2

# Test specific namespace
go run main.go -n <namespace> -o test.d2

# Run in silent mode
go run main.go -n <namespace> --quiet -o silent.d2

# Verify build after changes
go build -o k8sdd . && ./k8sdd -o /tmp/test.d2

# Run against live cluster (requires valid kubeconfig)
./k8sdd --all-namespaces -o output.d2
```

## Code Quality

```bash
# Format code (always run before committing)
go fmt ./...

# Run linter
golangci-lint run

# Check cyclomatic complexity
gocyclo -over 15 .
```

## Common Tasks

### Add New K8s Resource

1. Add type to model (`pkg/model/types.go`)
2. Add fetch function (`pkg/kube/fetch.go`)
3. Add rendering logic (`pkg/render/d2.go`)
4. Test with real cluster

### Add New CLI Flag

1. Add field to `RootOptions` struct (`cmd/root.go`)
2. Bind flag in `init()` function (`cmd/root.go`)
3. Use flag via `rootOptions.fieldName` in code
4. Test with `k8sdd --help`

### Fix High Complexity Function

If `gocyclo` reports complexity > 15:

1. Extract helper methods
2. Use early returns to reduce nesting
3. Split into smaller, focused functions

### Debug D2 Output

```bash
# Generate and view output
go run main.go -n test -o /tmp/out.d2 && cat /tmp/out.d2

# Validate D2 syntax
d2 /tmp/out.d2 /tmp/out.svg
```

## Frequently Modified Files

- `cmd/root.go` - Adding CLI flags
- `pkg/kube/fetch.go` - Adding K8s resource fetchers
- `pkg/render/d2.go` - Changing D2 output

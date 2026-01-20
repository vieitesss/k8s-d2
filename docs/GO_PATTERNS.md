# Go Patterns for k8sdd

Go coding patterns and style conventions specific to this project.

## Error Handling with Spinners

Wrap operations with spinner + separate error handling:

```go
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

Both `spinnerErr` (UI errors) and `err` (operation errors) must be checked.

## D2 ID Sanitization

D2 requires safe identifiers. Always sanitize before using as D2 ID:

```go
sanitizedID := sanitizeID(resourceName) // "my-service" -> "my_service"
```

Never use hyphens in D2 IDs - use `sanitizeID()` from `pkg/render/d2.go`.

## Label Matching Pattern

Service selectors → Workload labels. Selector is subset of labels, all keys must match:

```go
// See labelsMatch() in pkg/render/d2.go
// Used for connecting services to workloads
```

## Namespace Fetcher Pattern

Use type alias for resource fetchers:

```go
type namespaceFetcher func(context.Context, string, *model.Namespace) error

fetchers := []namespaceFetcher{
    c.fetchDeployments,
    c.fetchStatefulSets,
}
```

This allows sequential iteration through different resource types.

## RootOptions Pattern

All CLI flags use the `RootOptions` struct pattern (not global variables):

```go
// In cmd/root.go
type RootOptions struct {
    Namespace      string
    AllNamespaces  bool
    OutputFile     string
    // Add new flags here
}

// In init()
rootCmd.Flags().StringVarP(&rootOptions.Namespace, "namespace", "n", "", "description")

// In command code
if rootOptions.AllNamespaces {
    // use the flag
}
```

## Standard Library Usage

- `context.Context` - Always pass from cobra commands to client methods
- `io.Writer` - Used for rendering output (file or stdout)
- `strings.Builder` - Prefer for D2 syntax construction

## Don't Do This

- ❌ Don't add global variables - use `RootOptions` struct pattern
- ❌ Don't write to stderr during spinner operations (breaks UI)
- ❌ Don't use hyphens in D2 IDs (use `sanitizeID()`)
- ❌ Don't add flags without updating `RootOptions` in `cmd/root.go`
- ❌ Don't suppress klog output anywhere except `pkg/kube/client.go` init()

## Best Examples to Follow

When unsure about patterns, look at these files:

- `cmd/root.go` - RootOptions struct pattern for flags
- `pkg/kube/fetch.go` - namespaceFetcher type alias, sequential fetching
- `pkg/render/d2.go` - strings.Builder for text generation, helper methods
- `cmd/generate.go` - Spinner wrapper pattern, error propagation

## Cyclomatic Complexity

Keep functions under complexity 15. This threshold aligns with our project maintainability guidelines and the common default used by tools like `gocyclo`. If a function exceeds this:
- Extract helper methods
- Reduce nesting (early returns help)
- Split into smaller functions

Check complexity with: `gocyclo -over 15 .`

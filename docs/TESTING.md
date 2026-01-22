# Testing k8sdd

Manual testing procedures and verification steps.

## Manual Testing Flow

1. Build: `go build -o k8sdd .`
2. Run against test namespace: `./k8sdd -n <test-ns> -o test.d2`
3. Verify D2 syntax: `d2 test.d2 test.svg`
4. Check output has: namespaces, workloads with icons, services, connections

## Verification Checklists

### After Adding K8s Resource Support

- [ ] Resource appears in model (`pkg/model/types.go`)
- [ ] Fetch function exists (`pkg/kube/fetch.go`)
- [ ] Renderer handles it (`pkg/render/d2.go`)
- [ ] D2 output is valid (test with `d2` CLI)

### After Adding CLI Flag

- [ ] Flag defined in `RootOptions` struct
- [ ] Flag bound in `init()` function
- [ ] Flag accessed via `rootOptions.fieldName`
- [ ] Help text accurate (`k8sdd --help`)

### Quick Verification Build

After significant changes, verify the binary builds and runs:

```bash
go build -o k8sdd . && ./k8sdd -o /tmp/test.d2
```

## Testing Against Live Clusters

If you have access to a Kubernetes cluster (requires valid kubeconfig):

```bash
# Test all namespaces
./k8sdd --all-namespaces -o output.d2

# Test specific namespace
./k8sdd -n kube-system -o system.d2

# Test in quiet mode
./k8sdd -n default --quiet -o quiet.d2
```

## Edge Cases to Test

When adding new features, test these scenarios:

- Empty namespaces (no resources)
- Resources without labels
- Services without selectors
- Resources with special characters in names
- Very large clusters (many namespaces/resources)

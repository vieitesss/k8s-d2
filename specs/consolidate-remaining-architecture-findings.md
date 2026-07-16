# Spec: Consolidate the remaining render and normalization seams

## Problem Statement

As a maintainer of k8sdd, I still have two small sources of avoidable duplication and dead interface surface after collapsing validation onto exact rendering.

The Kubernetes StatefulSet PVC naming rule (`template-statefulset-ordinal`) is repeated by live-cluster PVC extraction, live-cluster volume-mount extraction, and fixture normalization. Those paths must remain identical for exact-render validation to compare live and fixture-derived topology reliably, but today their parity depends on three format strings staying synchronized by hand.

The D2 renderer also accepts a deprecated `gridColumns` constructor argument that it ignores because D2 now performs automatic layout. The CLI carries this value through rendering even though it cannot affect output, making the renderer interface and command plumbing advertise behavior that no longer exists.

The architecture review additionally identified duplicate connection derivation between rendering and validation. The validator-collapse work removes the validation-side derivation entirely, so introducing a new shared connection model would preserve abstraction for a second reader that no longer exists.

## Solution

Keep connection derivation inside the D2 renderer as the sole implementation of diagram connection rules; do not introduce a shared connection abstraction without a second consumer.

Centralize the Kubernetes-generated StatefulSet PVC name in one package-local helper and route live PVC extraction, live volume-mount extraction, and fixture normalization through it. This makes live and fixture topology parity structural rather than convention-based.

Narrow the D2 renderer constructor to accept only its output writer. Remove `gridColumns` from command options and renderer call sites while retaining the deprecated CLI flag at the Cobra flag boundary for compatibility. The flag remains accepted and documented as having no effect, but its value is not stored or passed into rendering.

## User Stories

1. As a maintainer, I want connection derivation to have one implementation in the D2 renderer, so that graph rules cannot drift between rendering and validation.
2. As a maintainer, I want to avoid a shared connection abstraction with only one consumer, so that the model does not gain speculative interface surface.
3. As a maintainer, I want service-to-workload connections to continue using the renderer's existing selector-matching behavior, so that this cleanup does not change topology output.
4. As a maintainer, I want entrypoint-to-service connections to continue excluding references to absent services, so that this cleanup does not introduce dangling D2 edges.
5. As a maintainer, I want workload-to-PVC connections to retain their existing grouping, ordering, and labels, so that storage diagrams remain stable.
6. As a maintainer, I want the generated StatefulSet PVC name rule defined once, so that Kubernetes naming parity cannot drift across code paths.
7. As a live-cluster user, I want StatefulSet PVC names to remain `template-statefulset-ordinal`, so that rendered storage nodes correspond to Kubernetes-created claims.
8. As a live-cluster user, I want StatefulSet volume mounts to reference those same generated PVC names, so that workload-to-storage edges point to rendered claims.
9. As a fixture-based validation user, I want synthesized StatefulSet PVCs to use the same naming rule as live extraction, so that exact-render comparison remains reliable.
10. As a fixture-based validation user, I want replicas and multiple volume claim templates to continue producing one generated name per template and ordinal, so that complete StatefulSet storage topology is represented.
11. As a fixture-based validation user, I want zero-replica StatefulSets to continue producing no generated PVC names, mounts, or synthesized PVCs, so that normalization reflects the requested replica count.
12. As a contributor, I want the PVC naming helper to remain internal to the Kubernetes package, so that an implementation detail does not become a public API.
13. As a contributor, I want the D2 renderer constructor to require only an output writer, so that its interface describes every input that can actually affect rendering.
14. As a CLI user, I want existing invocations containing `--grid-columns` to remain accepted during the deprecation period, so that removing dead internal plumbing does not break scripts prematurely.
15. As a CLI user, I want `--grid-columns` to remain clearly marked deprecated and ineffective, so that I can remove it from my scripts before a future breaking release.
16. As a maintainer, I want the deprecated flag value to stop flowing through command options and rendering functions, so that dead state is not carried through the application.
17. As a maintainer, I want all renderer call sites to use the narrowed constructor, so that no caller supplies meaningless layout configuration.
18. As a user generating D2 or SVG output, I want rendering output to remain byte-for-byte unchanged, so that this maintenance work causes no diagram churn.
19. As a reviewer, I want these changes limited to deleting dead connection and layout concepts and centralizing one naming rule, so that the behavior-preserving intent is easy to verify.
20. As a future maintainer, I want a shared connection seam introduced only when a real second consumer appears, so that any future abstraction is justified by current behavior rather than speculation.

## Implementation Decisions

- The validator-collapse change is a prerequisite and resolves the duplicate connection-derivation finding by deleting the validation-side `RelationshipDeriver` and granular connection validators.
- Connection derivation remains private to the D2 renderer. Service selector matching, entrypoint service resolution, PVC mount grouping, deterministic ordering, and D2 edge formatting remain unchanged.
- No `Connection` domain type, `Connections` model API, or other shared graph abstraction will be introduced. There is only one remaining consumer.
- A single package-local StatefulSet PVC naming helper will format a volume claim template name, StatefulSet name, and replica ordinal according to Kubernetes' generated claim naming convention.
- Live PVC-name extraction, live volume-mount extraction, and fixture PVC normalization will all call that helper.
- The helper will not perform validation or alter replica handling; callers retain their current iteration and data-shaping responsibilities.
- The D2 renderer constructor will accept only an output writer. Automatic D2 layout remains the only layout behavior.
- All production, validation, parity-test, command-test, and renderer-test constructor calls will use the narrowed renderer interface.
- The deprecated `--grid-columns` flag will remain registered and marked deprecated at the CLI boundary for backward compatibility, but its value will not be stored in root command options or passed to rendering.
- No rendering, normalization, fixture, model, or CLI output contract changes are intended.
- Removal of the deprecated CLI flag itself is deferred to the project's existing breaking-change policy; this spec only removes its dead internal plumbing.

## Testing Decisions

- Good tests assert observable behavior rather than helper calls or private implementation structure. Tests should prove live/fixture parity, stable rendered output, and CLI compatibility without asserting that a particular private helper was invoked.
- StatefulSet PVC naming will be covered at the existing normalization parity seam. Equivalent live-cluster and fixture inputs must produce matching workloads, PVC names, volume mounts, and rendered topology.
- Existing Kubernetes extraction tests will continue to cover generated names for replicas, volume claim templates, and mounts. Cases already represented there should be updated only as required by the refactor; no direct unit test of the package-local formatting helper is needed.
- Existing renderer tests remain the behavioral seam for service, entrypoint, and PVC connections. Their expected D2 output must remain unchanged, confirming that no shared connection abstraction or graph behavior change was introduced.
- Existing exact-render validator tests continue to prove that fixture-derived expectations and rendered output agree after connection derivation is removed from validation.
- Existing CLI tests will verify that `--grid-columns` remains registered, deprecated, and accepted while having no effect on generated output.
- Existing D2 and SVG generation tests will verify that the narrowed renderer constructor does not change either output path.
- Prior art is the repository's normalization parity suite, Kubernetes extraction tests, renderer golden-style assertions, exact-render validator tests, and root/generate command tests.
- Verification will run the Kubernetes normalization and extraction tests, renderer tests, validation tests, command tests, and then the full Go test suite.

## Out of Scope

- Reintroducing granular D2 validators or the validation-side relationship deriver.
- Adding a shared connection model or moving renderer graph rules into the domain model.
- Changing service selector matching, entrypoint resolution, PVC edge grouping, ordering, labels, or any rendered D2 syntax.
- Changing the Kubernetes StatefulSet PVC naming convention.
- Refactoring the broader StatefulSet extraction or normalization flows beyond routing generated names through one helper.
- Removing the deprecated `--grid-columns` CLI flag entirely.
- Adding a replacement layout option or configuring D2 layout.
- Changing fixture formats, Kubernetes API interactions, SVG generation, or Kroki behavior.

## Further Notes

- The connection-derivation finding needs no new implementation after validator collapse: deletion leaves one reader and one source of truth. Creating a shared seam at that point would add indirection without reducing duplication.
- The StatefulSet helper protects the parity invariant established by the fixture-normalization work: equivalent live and fixture resources must normalize to the same cluster model before exact-render validation.
- Retaining the deprecated flag only at the CLI boundary preserves compatibility while allowing the renderer and internal command flow to tell the truth about their inputs.
- These findings are intentionally consolidated because they complete the same architecture-review cleanup after validator collapse, while remaining behavior-preserving and small.

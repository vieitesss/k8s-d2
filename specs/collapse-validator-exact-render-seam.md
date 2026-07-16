# Spec: Collapse the D2 validator onto the exact-render seam

## Problem Statement

As a maintainer of k8sdd, I have to keep two independent implementations of the
same topology-to-D2 knowledge in sync by hand. The `render` package turns a
`Cluster` model into D2 text, and the `validation` package's `RelationshipDeriver`
plus its eight granular `Validate*` methods re-derive that same expected output a
second way to check it. Every change to how the diagram renders (a new node style,
a changed connection rule, an escaped identifier) forces a parallel edit in the
validator, and if the two drift the tests can pass while the diagram is wrong — or
fail while it is right. The re-derivation is ~480 lines that check nothing
`ValidateExactRender` doesn't already check whole.

## Solution

The `render` package becomes the single source of truth for D2 output. The
validator keeps only two checks: `ValidateSyntax` (structural sanity — balanced
braces, direction header) and `ValidateExactRender` (byte-for-byte comparison of
the actual D2 against `render.NewD2Renderer` output for the expected `Cluster`).
The `RelationshipDeriver` and the granular resource/label/connection/config/legend
matchers are deleted. Validation now crosses one seam, and the renderer is the only
place topology-to-D2 rules live.

## User Stories

1. As a maintainer, I want D2-shape validation to route through a single
   exact-render comparison, so that I never have to update a second derivation
   when the renderer changes.
2. As a maintainer, I want the renderer to be the only implementation of
   topology-to-D2 rules, so that renderer/deriver drift becomes impossible.
3. As a maintainer, I want the validation package interface to shrink to
   `ValidateSyntax` + `ValidateExactRender`, so that there is less surface to
   learn and maintain.
4. As a maintainer, I want the `RelationshipDeriver` and its test file removed,
   so that connection-graph logic exists in exactly one place.
5. As a maintainer, I want the granular `Validate*` matchers and their tests
   removed, so that the test suite stops asserting a subset of what exact-render
   already asserts.
6. As a Dagger integration test author, I want `TestD2Output_BasicFromEnv` and
   `TestD2Output_StorageFromEnv` to still validate live-cluster output against
   fixture-derived expectations, so that end-to-end parity coverage is preserved.
7. As a contributor debugging a validation failure, I want the exact-render
   mismatch error to name the first differing line, so that I can locate the
   regression quickly (existing `exactRenderMismatch` behavior is retained).
8. As a contributor, I want fixture parsing (`FixtureParser`) untouched, so that
   the expected-`Cluster` construction path is unaffected by this change.
9. As a reviewer, I want the diff to only remove code and adjust call sites,
   so that I can confirm no rendering behavior changed.

## Implementation Decisions

- **Modules removed:**
  - `internal/validation/relationships.go` (the `RelationshipDeriver`, `Connection`
    type, and the three `*Connections` derivation methods) and its test file
    `relationships_test.go`.
  - The granular methods on `D2Validator`: `ValidateLegendStructure`,
    `ValidateResources`, `ValidateWorkloadLabels`, `ValidateEntrypointConnections`,
    `ValidateServiceConnections`, `ValidatePVCConnections`, `ValidateConfigInfo`.
  - Their supporting helpers once they have no remaining callers:
    `sortConnections`, `containsD2Line`, `containsD2LinePrefix`,
    `extractNamespaceBlock`, `extractD2Block`.
- **Module retained and narrowed:** `D2Validator` keeps `NewD2Validator`,
  `ValidateSyntax`, and `ValidateExactRender`. The `deriver` field is dropped from
  the struct.
- **Interface after change:** `NewD2Validator(expected *model.Cluster, d2Output string)`
  returning a validator exposing exactly `ValidateSyntax() error` and
  `ValidateExactRender() error`.
- **Source of truth:** `render.NewD2Renderer(...).Render(expected)` is the only
  producer of expected D2; `ValidateExactRender` already invokes it. No renderer
  changes are required for this candidate.
- **Error reporting:** `exactRenderMismatch` (first-differing-line diagnostic) is
  kept as-is.
- **Call-site updates:** the integration tests in
  `internal/validation/integration_test.go` drop calls to the removed `Validate*`
  methods, keeping only `ValidateSyntax` and `ValidateExactRender`.
- **Out of this candidate:** the shared connection-derivation seam (candidate 2)
  and the StatefulSet PVC naming helper (candidate 3) are follow-ups that fold in
  naturally once the deriver is gone; they are not part of this spec.

## Testing Decisions

- **What a good test asserts here:** external behavior of the validator — that
  correct D2 passes and incorrect D2 fails with a useful message — not the internal
  derivation of expected connections. Exact-render is the strongest external
  assertion available: it compares the whole rendered artifact.
- **Modules tested:** `D2Validator` via its two retained methods, exercised by the
  existing env-driven integration tests (`TestD2Output_BasicFromEnv`,
  `TestD2Output_StorageFromEnv`) that feed live `k8sdd` output and validate against
  fixture-derived `Cluster` expectations.
- **Coverage to preserve:** the exact-render path must still catch resource,
  label, connection, config, and legend regressions — it does, because any such
  regression changes the rendered bytes and fails `ValidateExactRender`.
- **Prior art:** the existing `validator_test.go` exact-render cases and the
  `integration_test.go` env-driven flow. Retain the exact-render-focused tests;
  remove tests that only exercised the deleted matchers and deriver.
- **Regression guard:** run the full `internal/validation` package tests plus the
  renderer tests (`pkg/render`) after removal to confirm exact-render still passes
  on all existing fixtures.

## Out of Scope

- Any change to renderer output or the `render` package interface (including the
  dead `gridColumns` parameter — that is candidate 4).
- Extracting a shared `model.Connections` seam (candidate 2).
- Consolidating the StatefulSet PVC naming rule (candidate 3).
- Fixture format, `FixtureParser`, or `kube` normalization changes.
- New validation capabilities beyond syntax + exact render.

## Further Notes

- Deletion test rationale: removing the granular matchers concentrates all
  D2-shape checking in `ValidateExactRender` — complexity vanishes, coverage does
  not. That is the signal that the removed cluster was a pass-through.
- This lands in the repo's hot spot (render / validation / normalize; commits
  #78, #79, #70, #62), so the maintenance payoff is immediate.
- Sequencing: do this first; candidates 2 and 3 become smaller once the parallel
  derivation is gone.

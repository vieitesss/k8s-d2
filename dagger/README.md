# k8sdd Dagger Module

This module runs `k8sdd` against a dummy kind cluster populated from `test/fixtures/`.

It exposes two main functions:

- `run`: applies the fixtures and validates the generated D2 output.
- `fixture-image`: applies the fixtures and exports the final SVG image.

## Requirements

- `dagger` CLI
- Docker socket access, usually `/var/run/docker.sock`
- A reachable kind service address, for example `tcp://localhost:3000`

## Usage

List available functions:

```bash
dagger functions
```

Run the validation flow:

```bash
dagger call run \
  --docker-socket /var/run/docker.sock \
  --kind-svc tcp://localhost:3000
```

Export the fixture-backed SVG image:

```bash
dagger call fixture-image \
  --docker-socket /var/run/docker.sock \
  --kind-svc tcp://localhost:3000 \
  export --path ./cluster.svg
```

Export the SVG with storage resources included:

```bash
dagger call fixture-image \
  --docker-socket /var/run/docker.sock \
  --kind-svc tcp://localhost:3000 \
  --include-storage \
  export --path ./cluster-with-storage.svg
```

If you want to connect through an existing kubeconfig directory, add:

```bash
--kubeconfig file://$HOME/.kube
```

## Just Recipes

From `dagger/`:

```bash
just --list
just cluster
just validate
just image
just image-storage
```

The `cluster` recipe ensures a local kind cluster named `kind` exists.
The `validate`, `image`, and `image-storage` recipes automatically resolve the API server port from that cluster.

Defaults are hardcoded for local use:

- Docker socket: `/var/run/docker.sock`
- Kubeconfig: `file://$HOME/.kube`

The only optional parameter is the output filename:

```bash
just image
just image custom.svg
just image-storage
just image-storage custom-storage.svg
```

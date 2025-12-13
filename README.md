# k8s-d2

[![Go Report Card](https://goreportcard.com/badge/github.com/vieitesss/k8s-d2)](https://goreportcard.com/report/github.com/vieitesss/k8s-d2)
[![License](https://img.shields.io/github/license/vieitesss/k8s-d2)](LICENSE)
[![Release](https://img.shields.io/github/v/release/vieitesss/k8s-d2)](https://github.com/vieitesss/k8s-d2/releases)

A command-line tool that generates [D2](https://d2lang.com/) diagram files from Kubernetes cluster topology. Visualize your cluster's namespaces, workloads, services, and their relationships as code.

## Features

- Generate D2 diagrams from live Kubernetes clusters
- Visualize workloads (Deployments, StatefulSets, DaemonSets) with distinct icons
- Map service-to-workload relationships
- Filter by namespace or view entire cluster
- Track ConfigMaps and Secrets per namespace
- Customizable grid layout for namespace organization
- Output to file or stdout for pipeline integration

## Installation

### Homebrew (macOS/Linux)

```bash
brew install vieitesss/tap/k8s-d2
```

### Go Install

```bash
go install github.com/vieitesss/k8s-d2@latest
```

### Pre-compiled Binaries

Download the latest release for your platform from the [releases page](https://github.com/vieitesss/k8s-d2/releases).

**Linux (amd64):**
```bash
curl -LO https://github.com/vieitesss/k8s-d2/releases/latest/download/k8s-d2_Linux_x86_64.tar.gz
tar -xzf k8s-d2_Linux_x86_64.tar.gz
sudo mv k8sdd /usr/local/bin/
```

**macOS (Apple Silicon):**
```bash
curl -LO https://github.com/vieitesss/k8s-d2/releases/latest/download/k8s-d2_Darwin_arm64.tar.gz
tar -xzf k8s-d2_Darwin_arm64.tar.gz
sudo mv k8sdd /usr/local/bin/
```

**Windows:**
Download the `.zip` file from releases and add the extracted binary to your PATH.

### Build from Source

```bash
git clone https://github.com/vieitesss/k8s-d2.git
cd k8s-d2
go build -o k8sdd .
```

## Quick Start

Generate a diagram from your current Kubernetes context:

```bash
k8sdd -o cluster.d2
```

Render the diagram with D2:

```bash
d2 cluster.d2 cluster.svg
```

## Usage

### Basic Commands

```bash
# Output to stdout
k8sdd

# Save to file
k8sdd -o cluster.d2

# Visualize specific namespace
k8sdd -n monitoring -o monitoring.d2

# Include all namespaces (including system namespaces)
k8sdd --all-namespaces -o full-cluster.d2

# Use custom kubeconfig
k8sdd --kubeconfig ~/.kube/prod-config -o prod.d2
```

### Layout Options

```bash
# Control namespace grid layout (default: 3 columns)
k8sdd --grid-columns 2 -o wide-layout.d2

# Single column layout
k8sdd --grid-columns 1 -o vertical.d2
```

### Advanced Usage

```bash
# Include storage layer (PVCs, StorageClasses)
k8sdd --include-storage -o storage.d2

# Combine options
k8sdd --all-namespaces --include-storage --grid-columns 4 -o complete.d2
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--kubeconfig` | | `~/.kube/config` | Path to kubeconfig file |
| `--namespace` | `-n` | | Filter by specific namespace |
| `--all-namespaces` | `-A` | `false` | Include system namespaces |
| `--output` | `-o` | stdout | Output file path |
| `--grid-columns` | | `3` | Number of columns for namespace layout |
| `--include-storage` | | `false` | Include PVCs and StorageClasses |

## Output Format

The tool generates D2 syntax representing your cluster topology:

- **Namespaces**: Containers with light gray fill (`#f0f0f0`)
- **Workloads**: Nodes with type-specific icons
  - Deployments: ●
  - StatefulSets: ◉
  - DaemonSets: ◈
- **Services**: Blue-filled nodes (`#cce5ff`) showing service type
- **Config/Secrets**: Yellow-filled summary node (`#ffffcc`)
- **Connections**: Service-to-workload relationships via selectors

See [examples/sample-output.d2](examples/sample-output.d2) for reference output.

## Requirements

- Kubernetes cluster access via kubeconfig
- Valid KUBECONFIG or `~/.kube/config` file
- [D2](https://d2lang.com/) for rendering diagrams (optional, for visualization)

## Project Structure

```
cmd/
  root.go       # CLI setup and global flags
  generate.go   # Main generation command logic
pkg/
  kube/
    client.go   # Kubernetes client initialization
    fetch.go    # Resource fetching and filtering
  model/
    types.go    # Internal graph representation
  render/
    d2.go       # D2 syntax generation
main.go         # Application entry point
```

## Roadmap

- [x] Phase 1: Basic topology (namespaces, workloads, services)
- [ ] Phase 2: Storage layer (PVCs, volumes, StorageClasses)
- [ ] Phase 3: Network layer (Ingress, NetworkPolicies)
- [ ] Custom styling themes
- [ ] Interactive filtering and drill-down

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [client-go](https://github.com/kubernetes/client-go) - Kubernetes API client
- [D2](https://d2lang.com/) - Declarative diagramming language

// Dagger CI module for k8s-d2
//
// This module provides CI/CD testing for k8s-d2 by deploying test fixtures
// to a local kind cluster and validating D2 output.

package main

import (
	"context"
	"dagger/dagger/internal/dagger"
	"fmt"
)

type Dagger struct {
	// Defaults to the root of the repository.
	Src *dagger.Directory
}

func New(
	// +defaultPath="/"
	src *dagger.Directory,
) *Dagger {
	return &Dagger{
		Src: src,
	}
}

// Use when the kind cluster needs to be created.
func (m *Dagger) Run(
	ctx context.Context,

	// Docker socket path
	dockerSocket *dagger.Socket,

	// Your already created Kind cluster address.
	// Example: `tcp://localhost:3000`
	kindSvc *dagger.Service,

	// Directory containing kubeconfig files for your cluster.
	// Example: `$HOME/.kube`
	// +optional
	kubeconfig *dagger.Directory,
) (string, error) {
	var kindCtr *dagger.Container
	var err error

	if kubeconfig != nil {
		kindCtr, err = m.KindFromService(ctx, kindSvc, kubeconfig)
	} else {
		kindCtr = m.KindFromModule(dockerSocket, kindSvc)
	}
	if err != nil {
		return "", fmt.Errorf("failed to create kind container: %w", err)
	}

	return m.test(ctx, kindCtr)
}

func (m *Dagger) test(ctx context.Context, kindCtr *dagger.Container) (string, error) {
	kindBinaryCtr := m.build(kindCtr)

	fixturesDir := m.Src.Directory("test/fixtures")

	kindBinFixCtr, err := ApplyFixtures(ctx, kindBinaryCtr, fixturesDir, true)
	if err != nil {
		return "", fmt.Errorf("failed to apply fixtures: %w", err)
	}

	basicOutput, err := m.runK8sD2(ctx, kindBinFixCtr, false)
	if err != nil {
		return "", fmt.Errorf("k8s-d2 execution failed (basic): %w", err)
	}

	basicValidator := NewD2Validator(basicOutput)
	if err := basicValidator.ValidateSyntax(); err != nil {
		return "", fmt.Errorf("syntax validation failed (basic): %w", err)
	}
	if err := basicValidator.ValidateBasicTopology(); err != nil {
		return "", fmt.Errorf("topology validation failed (basic): %w", err)
	}
	if err := basicValidator.ValidateLabels(); err != nil {
		return "", fmt.Errorf("label validation failed (basic): %w", err)
	}
	if err := basicValidator.ValidateConnections(); err != nil {
		return "", fmt.Errorf("connection validation failed (basic): %w", err)
	}

	storageOutput, err := m.runK8sD2(ctx, kindBinFixCtr, true)
	if err != nil {
		return "", fmt.Errorf("k8s-d2 execution failed (storage): %w", err)
	}

	storageValidator := NewD2Validator(storageOutput)
	if err := storageValidator.ValidateSyntax(); err != nil {
		return "", fmt.Errorf("syntax validation failed (storage): %w", err)
	}
	if err := storageValidator.ValidateBasicTopology(); err != nil {
		return "", fmt.Errorf("topology validation failed (storage): %w", err)
	}
	if err := storageValidator.ValidateStorage(); err != nil {
		return "", fmt.Errorf("storage validation failed: %w", err)
	}

	return "All tests passed! ✓\n- Basic topology validated\n- Storage layer validated\n- D2 syntax correct\n- All resources present", nil
}

// build compiles k8s-d2 binary
func (m *Dagger) build(ctr *dagger.Container) *dagger.Container {
	binary := dag.Container().
		From("golang:1.24").
		WithDirectory("/src", m.Src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build")).
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec([]string{"go", "build", "-o", "k8sdd", "."}).
		File("/src/k8sdd")

	return ctr.
		WithFile("/usr/local/bin/k8sdd", binary).
		WithExec([]string{"chmod", "+x", "/usr/local/bin/k8sdd"})
}

// runK8sD2 executes k8s-d2 against the cluster and returns D2 output
func (m *Dagger) runK8sD2(
	ctx context.Context,

	// Container with k8sdd binary
	ctr *dagger.Container,

	includeStorage bool,
) (string, error) {
	ctr = ctr.
		WithExec([]string{"mkdir", "-p", "/output"}).
		WithWorkdir("/output")

	args := []string{"k8sdd", "-n", "k8s-d2-test", "-o", "test.d2"}
	if includeStorage {
		args = []string{"k8sdd", "-n", "k8s-d2-test", "--include-storage", "-o", "/output/test.d2"}
	}

	file := ctr.
		WithExec(args).
		File("/output/test.d2")

	output, err := file.Contents(ctx)
	
	if err != nil {
		return "", err
	}

	return output, nil
}

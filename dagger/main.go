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

	// Generate D2 outputs from real cluster
	basicOutput, err := m.runK8sD2(ctx, kindBinFixCtr, false)
	if err != nil {
		return "", fmt.Errorf("k8s-d2 execution failed (basic): %w", err)
	}

	storageOutput, err := m.runK8sD2(ctx, kindBinFixCtr, true)
	if err != nil {
		return "", fmt.Errorf("k8s-d2 execution failed (storage): %w", err)
	}

	quietOutput, err := m.runK8sD2Quiet(ctx, kindBinFixCtr, false)
	if err != nil {
		return "", fmt.Errorf("k8s-d2 execution failed (quiet mode): %w", err)
	}

	// Run Go tests to validate the D2 outputs
	testOutput, err := m.runValidationTests(ctx, basicOutput, storageOutput, quietOutput)
	if err != nil {
		return "", fmt.Errorf("validation tests failed: %w", err)
	}

	return fmt.Sprintf("All tests passed! ✓\n- Basic topology validated\n- Storage layer validated\n- D2 syntax correct\n- All resources present\n- Quiet flag validated\n\nTest output:\n%s", testOutput), nil
}

func (m *Dagger) Base() *dagger.Container {
	return dag.Container().
		From("golang:1.24").
		WithDirectory("/src", m.Src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build"))
}

// runValidationTests runs Go tests to validate D2 outputs
func (m *Dagger) runValidationTests(
	ctx context.Context,
	basicOutput string,
	storageOutput string,
	quietOutput string,
) (string, error) {
	testCtr := m.Base().
		WithEnvVariable("D2_OUTPUT_BASIC", basicOutput).
		WithEnvVariable("D2_OUTPUT_STORAGE", storageOutput).
		WithEnvVariable("D2_OUTPUT_QUIET", quietOutput).
		WithExec([]string{"go", "test", "-v", "./pkg/validation/..."})

	output, err := testCtr.Stdout(ctx)
	if err != nil {
		return "", err
	}

	return output, nil
}

// build compiles k8s-d2 binary
func (m *Dagger) build(ctr *dagger.Container) *dagger.Container {
	binary := m.Base().
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

	outputFile := "/output/test.d2"
	args := []string{"k8sdd", "-n", "k8s-d2-test", "-o", outputFile}
	if includeStorage {
		args = []string{"k8sdd", "-n", "k8s-d2-test", "--include-storage", "-o", outputFile}
	}

	file := ctr.WithExec(args).File(outputFile)

	output, err := file.Contents(ctx)
	if err != nil {
		return "", err
	}

	return output, nil
}

// runK8sD2Quiet executes k8s-d2 with --quiet flag, validates no logs are emitted, and returns D2 output
func (m *Dagger) runK8sD2Quiet(
	ctx context.Context,

	// Container with k8sdd binary
	ctr *dagger.Container,

	includeStorage bool,
) (string, error) {
	ctr = ctr.
		WithExec([]string{"mkdir", "-p", "/output"}).
		WithWorkdir("/output")

	outputFile := "/output/test-quiet.d2"
	stdoutFile := "/output/stdout.log"
	stderrFile := "/output/stderr.log"
	args := []string{"sh", "-c"}

	var cmd string
	if includeStorage {
		cmd = fmt.Sprintf("k8sdd -n k8s-d2-test --include-storage -o %s --quiet > %s 2> %s", outputFile, stdoutFile, stderrFile)
	} else {
		cmd = fmt.Sprintf("k8sdd -n k8s-d2-test -o %s --quiet > %s 2> %s", outputFile, stdoutFile, stderrFile)
	}
	args = append(args, cmd)

	execCtr := ctr.WithExec(args)

	// Check stdout for unwanted output.
	stdout, err := execCtr.File(stdoutFile).Contents(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to read stdout: %w", err)
	}

	if stdout != "" {
		return "", fmt.Errorf("quiet mode test failed: stdout should be empty but contains: %s", stdout)
	}

	// Get the D2 output
	output, err := execCtr.File(outputFile).Contents(ctx)
	if err != nil {
		return "", err
	}

	return output, nil
}

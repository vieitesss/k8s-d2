// Dagger CI module for k8s-d2
//
// This module provides CI/CD testing for k8s-d2 by deploying test fixtures
// to a local kind cluster and validating D2 output.

package main

import (
	"context"
	"dagger/dagger/internal/dagger"
	"dagger/dagger/k8sddcmd"
	"fmt"
	"strings"
)

const (
	fixtureNamespace = "k8s-d2-test"
	outputDir        = "/output"
	basicOutputFile  = outputDir + "/test.d2"
	quietOutputFile  = outputDir + "/test-quiet.d2"
	imageOutputFile  = outputDir + "/test.svg"
	goModCacheDir    = "/go/pkg/mod"
	goBuildCacheDir  = "/root/.cache/go-build"
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

	// Reuse the existing fixture namespace instead of deleting it first.
	// +optional
	reuseNamespace bool,
) (string, error) {
	kindCtr, err := m.fixtureCluster(ctx, dockerSocket, kindSvc, kubeconfig, reuseNamespace)
	if err != nil {
		return "", err
	}

	return m.test(ctx, kindCtr)
}

// FixtureImage returns the SVG generated from the fixture-backed kind cluster.
func (m *Dagger) FixtureImage(
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

	// Reuse the existing fixture namespace instead of deleting it first.
	// +optional
	reuseNamespace bool,

	// Include the storage layer in the generated diagram.
	// +optional
	includeStorage bool,
) (*dagger.File, error) {
	kindCtr, err := m.fixtureCluster(ctx, dockerSocket, kindSvc, kubeconfig, reuseNamespace)
	if err != nil {
		return nil, err
	}

	return m.runK8sD2Image(kindCtr, includeStorage), nil
}

func (m *Dagger) fixtureCluster(
	ctx context.Context,
	dockerSocket *dagger.Socket,
	kindSvc *dagger.Service,
	kubeconfig *dagger.Directory,
	reuseNamespace bool,
) (*dagger.Container, error) {
	kindCtr, err := m.kindContainer(ctx, dockerSocket, kindSvc, kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kind container: %w", err)
	}

	kindBinaryCtr := m.build(kindCtr)
	fixturesDir := m.Src.Directory("test/fixtures")

	kindBinFixCtr, err := ApplyFixtures(ctx, kindBinaryCtr, fixturesDir, fixtureNamespace, true, reuseNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to apply fixtures: %w", err)
	}

	return kindBinFixCtr, nil
}

func (m *Dagger) kindContainer(
	ctx context.Context,
	dockerSocket *dagger.Socket,
	kindSvc *dagger.Service,
	kubeconfig *dagger.Directory,
) (*dagger.Container, error) {
	if kubeconfig != nil {
		return m.KindFromService(ctx, kindSvc, kubeconfig)
	}

	return m.KindFromModule(dockerSocket, kindSvc), nil
}

func (m *Dagger) test(ctx context.Context, kindCtr *dagger.Container) (string, error) {
	// Generate D2 outputs from real cluster
	basicOutput, err := m.runK8sD2(ctx, kindCtr, false)
	if err != nil {
		return "", fmt.Errorf("k8s-d2 execution failed (basic): %w", err)
	}

	storageOutput, err := m.runK8sD2(ctx, kindCtr, true)
	if err != nil {
		return "", fmt.Errorf("k8s-d2 execution failed (storage): %w", err)
	}

	quietOutput, err := m.runK8sD2Quiet(ctx, kindCtr, false)
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

func (m *Dagger) BaseContainer() *dagger.Container {
	return dag.Container().
		From("golang:1.24").
		WithDirectory("/src", m.Src).
		WithWorkdir("/src").
		WithEnvVariable("GOMODCACHE", goModCacheDir).
		WithEnvVariable("GOCACHE", goBuildCacheDir).
		WithMountedCache(goModCacheDir, dag.CacheVolume("go-mod")).
		WithMountedCache(goBuildCacheDir, dag.CacheVolume("go-build"))
}

// runValidationTests runs Go tests to validate D2 outputs
func (m *Dagger) runValidationTests(
	ctx context.Context,
	basicOutput string,
	storageOutput string,
	quietOutput string,
) (string, error) {
	testCtr := m.BaseContainer().
		WithEnvVariable("D2_OUTPUT_BASIC", basicOutput).
		WithEnvVariable("D2_OUTPUT_STORAGE", storageOutput).
		WithEnvVariable("D2_OUTPUT_QUIET", quietOutput).
		WithExec([]string{"go", "test", "-v", "./internal/validation/..."})

	output, err := testCtr.Stdout(ctx)
	if err != nil {
		return "", err
	}

	return output, nil
}

// build compiles k8s-d2 binary
func (m *Dagger) build(ctr *dagger.Container) *dagger.Container {
	binary := m.BaseContainer().
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
	ctr = withOutputDir(ctr)
	file := ctr.WithExec(k8sddcmd.DiagramArgs(fixtureNamespace, basicOutputFile, includeStorage, false)).File(basicOutputFile)

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
	ctr = withOutputDir(ctr)

	stdoutFile := outputDir + "/stdout.log"
	stderrFile := outputDir + "/stderr.log"
	args := append(k8sddcmd.DiagramArgs(fixtureNamespace, quietOutputFile, includeStorage, false), "--quiet")
	cmd := fmt.Sprintf("%s > %s 2> %s", strings.Join(args, " "), stdoutFile, stderrFile)

	execCtr := ctr.WithExec([]string{"sh", "-c", cmd})

	// Check stdout for unwanted output.
	stdout, err := execCtr.File(stdoutFile).Contents(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to read stdout: %w", err)
	}

	if stdout != "" {
		return "", fmt.Errorf("quiet mode test failed: stdout should be empty but contains: %s", stdout)
	}

	// Check stderr for unwanted output.
	stderr, err := execCtr.File(stderrFile).Contents(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to read stderr: %w", err)
	}

	if stderr != "" {
		return "", fmt.Errorf("quiet mode test failed: stderr should be empty but contains: %s", stderr)
	}

	// Get the D2 output
	output, err := execCtr.File(quietOutputFile).Contents(ctx)
	if err != nil {
		return "", err
	}

	return output, nil
}

// runK8sD2Image executes k8sdd against the cluster and returns the SVG artifact.
func (m *Dagger) runK8sD2Image(
	ctr *dagger.Container,
	includeStorage bool,
) *dagger.File {
	ctr = withOutputDir(ctr)

	return ctr.WithExec(k8sddcmd.DiagramArgs(fixtureNamespace, imageOutputFile, includeStorage, true)).File(imageOutputFile)
}

func withOutputDir(ctr *dagger.Container) *dagger.Container {
	return ctr.
		WithExec([]string{"mkdir", "-p", outputDir}).
		WithWorkdir(outputDir)
}

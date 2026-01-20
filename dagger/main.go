// Dagger CI module for k8s-d2
package main

import (
	"context"
	"dagger/dagger/internal/dagger"
	"fmt"
)

type Dagger struct {
	Src          *dagger.Directory
	GoModCache   *dagger.Directory
	GoBuildCache *dagger.Directory
}

func New(
	// +defaultPath="/"
	src *dagger.Directory,
) *Dagger {
	return &Dagger{Src: src}
}

// Run executes tests. 
// Locally: returns an empty directory to avoid Dagger cache-retrieve errors.
// CI: returns the populated go-build cache directory for GitHub Actions to save.
func (m *Dagger) Run(
	ctx context.Context,
	dockerSocket *dagger.Socket,
	kindSvc *dagger.Service,
	// +optional
	kubeconfig *dagger.Directory,
	// +optional
	goModCache *dagger.Directory,
	// +optional
	goBuildCache *dagger.Directory,
) *dagger.Directory {
	m.GoModCache = goModCache
	m.GoBuildCache = goBuildCache

	var kindCtr *dagger.Container
	var err error

	if kubeconfig != nil {
		kindCtr, err = m.KindFromService(ctx, kindSvc, kubeconfig)
		if err != nil {
			panic(fmt.Errorf("failed to create kind container: %w", err))
		}
	} else {
		kindCtr = m.KindFromModule(dockerSocket, kindSvc)
	}

	// Run tests and get the build container
	buildCtr := m.test(ctx, kindCtr)

	// If no cache flags were passed (Local Mode), return an empty directory.
	if m.GoBuildCache == nil {
		return dag.Directory()
	}

	// CI MODE: Extract only the build cache.
	// We skip go-mod because thousands of tiny files cause GitHub I/O hangs.
	buildDir := buildCtr.Directory("/root/.cache/go-build")
	_, _ = buildDir.Sync(ctx)

	return dag.Directory().WithDirectory("go-build", buildDir)
}

func (m *Dagger) test(ctx context.Context, kindCtr *dagger.Container) *dagger.Container {
	kindBinaryCtr, buildCtr := m.build(kindCtr)
	fixturesDir := m.Src.Directory("test/fixtures")

	kindBinFixCtr, err := ApplyFixtures(ctx, kindBinaryCtr, fixturesDir, true)
	if err != nil {
		panic(fmt.Errorf("failed to apply fixtures: %w", err))
	}

	basic, _ := m.runK8sD2(ctx, kindBinFixCtr, false)
	storage, _ := m.runK8sD2(ctx, kindBinFixCtr, true)
	quiet, _ := m.runK8sD2Quiet(ctx, kindBinFixCtr, false)

	testOutput, err := m.runValidationTests(ctx, basic, storage, quiet)
	if err != nil {
		panic(fmt.Errorf("validation tests failed: %w", err))
	}

	fmt.Printf("All tests passed!\n%s\n", testOutput)

	return buildCtr
}

func (m *Dagger) BaseContainer() *dagger.Container {
	ctr := dag.Container().
		From("golang:1.24").
		WithDirectory("/src", m.Src).
		WithWorkdir("/src")

	// Even if we don't export go-mod, we still use the mount for the current run
	if m.GoModCache != nil {
		return ctr.
			WithMountedDirectory("/go/pkg/mod", m.GoModCache).
			WithMountedDirectory("/root/.cache/go-build", m.GoBuildCache)
	}

	return ctr.
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build"))
}

func (m *Dagger) build(ctr *dagger.Container) (*dagger.Container, *dagger.Container) {
	buildCtr := m.BaseContainer().
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec([]string{"go", "build", "-o", "k8sdd", "."})

	binary := buildCtr.File("/src/k8sdd")
	kindCtr := ctr.
		WithFile("/usr/local/bin/k8sdd", binary).
		WithExec([]string{"chmod", "+x", "/usr/local/bin/k8sdd"})

	return kindCtr, buildCtr
}

func (m *Dagger) runValidationTests(ctx context.Context, b, s, q string) (string, error) {
	return m.BaseContainer().
		WithEnvVariable("D2_OUTPUT_BASIC", b).
		WithEnvVariable("D2_OUTPUT_STORAGE", s).
		WithEnvVariable("D2_OUTPUT_QUIET", q).
		WithExec([]string{"go", "test", "-v", "./internal/validation/..."}).
		Stdout(ctx)
}

func (m *Dagger) runK8sD2(ctx context.Context, ctr *dagger.Container, storage bool) (string, error) {
	out := "/output/test.d2"
	args := []string{"k8sdd", "diagram", "-n", "k8s-d2-test", "-o", out}
	if storage {
		args = append(args, "--include-storage")
	}
	return ctr.WithExec([]string{"mkdir", "-p", "/output"}).WithExec(args).File(out).Contents(ctx)
}

func (m *Dagger) runK8sD2Quiet(ctx context.Context, ctr *dagger.Container, storage bool) (string, error) {
	out := "/output/test-quiet.d2"
	cmd := fmt.Sprintf("k8sdd diagram -n k8s-d2-test -o %s --quiet > /dev/null 2>&1", out)
	return ctr.WithExec([]string{"mkdir", "-p", "/output"}).WithExec([]string{"sh", "-c", cmd}).File(out).Contents(ctx)
}
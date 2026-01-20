package main

import (
	"context"
	"dagger/dagger/internal/dagger"
	"fmt"
)

type Dagger struct {
	Src          *dagger.Directory
	GoBuildCache *dagger.Directory
}

func New(
	// +defaultPath="/"
	src *dagger.Directory,
	// Moving this to the constructor ensures the CLI maps the host path correctly
	// +optional
	goBuildDir *dagger.Directory,
) *Dagger {
	return &Dagger{
		Src:          src,
		GoBuildCache: goBuildDir,
	}
}

func (m *Dagger) Run(
	ctx context.Context,
	dockerSocket *dagger.Socket,
	kindSvc *dagger.Service,
	// +optional
	kubeconfig *dagger.Directory,
) *dagger.Directory {
	var kindCtr *dagger.Container
	var err error

	// We bind the service to "localhost" so kubectl/k8sdd can reach it inside the container
	if kubeconfig != nil {
		kindCtr, err = m.KindFromService(ctx, kindSvc, kubeconfig)
		if err != nil {
			panic(err)
		}
	} else {
		// Ensure your KindFromModule also uses WithServiceBinding if it hits localhost
		kindCtr = m.KindFromModule(dockerSocket, kindSvc).
			WithServiceBinding("localhost", kindSvc)
	}

	// Run tests and capture the final container state
	buildCtr := m.test(ctx, kindCtr, kindSvc)

	// CI MODE: Return only the build cache for export to host
	if m.GoBuildCache != nil {
		return dag.Directory().
			WithDirectory("go-build", buildCtr.Directory("/root/.cache/go-build"))
	}

	return dag.Directory()
}

func (m *Dagger) BaseContainer(kindSvc *dagger.Service) *dagger.Container {
	ctr := dag.Container().
		From("golang:1.24").
		WithDirectory("/src", m.Src).
		WithWorkdir("/src").
		// ALWAYS use internal CacheVolume for modules (Safe & Fast)
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod"))

	// Bind the Kind service so Go tests can reach the cluster at localhost:3000
	if kindSvc != nil {
		ctr = ctr.WithServiceBinding("localhost", kindSvc)
	}

	if m.GoBuildCache != nil {
		return ctr.WithMountedDirectory("/root/.cache/go-build", m.GoBuildCache)
	}

	return ctr.WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build"))
}

func (m *Dagger) test(ctx context.Context, kindCtr *dagger.Container, kindSvc *dagger.Service) *dagger.Container {
	kindBinaryCtr, buildCtr := m.build(kindCtr, kindSvc)
	fixturesDir := m.Src.Directory("test/fixtures")

	kindBinFixCtr, err := ApplyFixtures(ctx, kindBinaryCtr, fixturesDir, true)
	if err != nil {
		panic(err)
	}

	b, _ := m.runK8sD2(ctx, kindBinFixCtr, false)
	s, _ := m.runK8sD2(ctx, kindBinFixCtr, true)
	q, _ := m.runK8sD2Quiet(ctx, kindBinFixCtr, false)

	out, err := m.runValidationTests(ctx, b, s, q, kindSvc)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Validation passed: %s\n", out)

	return buildCtr
}

func (m *Dagger) build(ctr *dagger.Container, kindSvc *dagger.Service) (*dagger.Container, *dagger.Container) {
	buildCtr := m.BaseContainer(kindSvc).
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec([]string{"go", "build", "-o", "k8sdd", "."})

	binary := buildCtr.File("/src/k8sdd")
	kindCtr := ctr.
		WithFile("/usr/local/bin/k8sdd", binary).
		WithExec([]string{"chmod", "+x", "/usr/local/bin/k8sdd"})

	return kindCtr, buildCtr
}

func (m *Dagger) runValidationTests(ctx context.Context, b, s, q string, kindSvc *dagger.Service) (string, error) {
	return m.BaseContainer(kindSvc).
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
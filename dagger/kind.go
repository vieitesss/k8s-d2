package main

import (
	"context"
	"dagger/dagger/internal/dagger"
	"fmt"
)

func (m *Dagger) KindFromModule(
	kindSocket *dagger.Socket,
	kindSvc *dagger.Service,
) *dagger.Container {
	return dag.Kind(kindSocket, kindSvc, dagger.KindOpts{
		Version: "v1_33",
	}).Container()
}

func (m *Dagger) KindFromService(
	ctx context.Context,
	kindSvc *dagger.Service,
	kubeconfig *dagger.Directory,
) (*dagger.Container, error) {
	ports, err := kindSvc.Ports(ctx)
	if err != nil {
		return nil, err
	}

	port, err := ports[0].Port(ctx)
	if err != nil {
		return nil, err
	}

	return dag.Container().
		From("alpine/k8s:1.31.0").
		WithMountedDirectory("/root/.kube", kubeconfig).
		WithServiceBinding("localhost", kindSvc).
		WithExec([]string{
			"kubectl", "config",
			"set-cluster",
			"kind-kind",
			fmt.Sprintf("--server=https://localhost:%d", port),
		}).
		WithExec([]string{"kubectl", "cluster-info"}).
		Sync(ctx)
}

package main

import (
	"context"
	"dagger/dagger/internal/dagger"
	"fmt"
)

// ApplyFixtures applies all test fixtures to the kind cluster in the correct order
func ApplyFixtures(
	ctx context.Context,
	kindContainer *dagger.Container,
	fixturesDir *dagger.Directory,
	namespace string,
	includeStorage bool,
	reuseNamespace bool,
	includeVisual bool,
) (*dagger.Container, error) {
	var err error
	fixtureNamespaces := []string{namespace}
	if includeVisual {
		fixtureNamespaces = append(fixtureNamespaces, "payments", "observability")
	}

	if !reuseNamespace {
		for _, fixtureNamespace := range fixtureNamespaces {
			kindContainer, err = kindContainer.
				WithExec([]string{"kubectl", "delete", "namespace", fixtureNamespace, "--ignore-not-found=true", "--wait=true", "--timeout=60s"}).
				Sync(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to clean up namespace %s: %w", fixtureNamespace, err)
			}
		}
	}

	// Apply base fixtures (sorted by filename to ensure correct order)
	baseFixtures := []string{
		"base/01-namespace.yaml",
		"base/02-configmaps-secrets.yaml",
		"base/03-deployments.yaml",
		"base/04-statefulsets.yaml",
		"base/05-daemonsets.yaml",
		"base/06-services.yaml",
	}

	for _, fixture := range baseFixtures {
		file := fixturesDir.File(fixture)
		kindContainer, err = kindContainer.
			WithMountedFile(fmt.Sprintf("/fixtures/%s", fixture), file).
			WithExec([]string{"kubectl", "apply", "-f", fmt.Sprintf("/fixtures/%s", fixture)}).
			Sync(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to apply %s: %w", fixture, err)
		}
	}

	if includeVisual {
		visualFixtures := []string{
			"visual/01-namespaces.yaml",
			"visual/02-configmaps-secrets.yaml",
			"visual/03-deployments.yaml",
			"visual/04-statefulsets.yaml",
			"visual/05-daemonsets.yaml",
			"visual/06-services.yaml",
			"visual/07-pvcs.yaml",
		}

		for _, fixture := range visualFixtures {
			file := fixturesDir.File(fixture)
			kindContainer, err = kindContainer.
				WithMountedFile(fmt.Sprintf("/fixtures/%s", fixture), file).
				WithExec([]string{"kubectl", "apply", "-f", fmt.Sprintf("/fixtures/%s", fixture)}).
				Sync(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to apply %s: %w", fixture, err)
			}
		}
	}

	// Apply storage fixtures if requested
	if includeStorage {
		storageFixtures := []string{
			"storage/01-storageclass.yaml",
			"storage/02-pvcs.yaml",
		}

		for _, fixture := range storageFixtures {
			file := fixturesDir.File(fixture)
			kindContainer, err = kindContainer.
				WithMountedFile(fmt.Sprintf("/fixtures/%s", fixture), file).
				WithExec([]string{"kubectl", "apply", "-f", fmt.Sprintf("/fixtures/%s", fixture)}).
				Sync(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to apply %s: %w", fixture, err)
			}
		}
	}

	readyChecks := [][]string{
		{"kubectl", "rollout", "status", "deployment/web-frontend", "-n", namespace, "--timeout=120s"},
		{"kubectl", "rollout", "status", "deployment/api-backend", "-n", namespace, "--timeout=120s"},
		{"kubectl", "rollout", "status", "statefulset/database", "-n", namespace, "--timeout=120s"},
		{"kubectl", "rollout", "status", "daemonset/log-collector", "-n", namespace, "--timeout=120s"},
	}
	if includeVisual {
		readyChecks = append(readyChecks,
			[]string{"kubectl", "rollout", "status", "deployment/payments-api", "-n", "payments", "--timeout=120s"},
			[]string{"kubectl", "rollout", "status", "deployment/payments-worker", "-n", "payments", "--timeout=120s"},
			[]string{"kubectl", "rollout", "status", "statefulset/redis-cache", "-n", "payments", "--timeout=120s"},
			[]string{"kubectl", "rollout", "status", "deployment/grafana", "-n", "observability", "--timeout=120s"},
			[]string{"kubectl", "rollout", "status", "statefulset/prometheus", "-n", "observability", "--timeout=120s"},
			[]string{"kubectl", "rollout", "status", "daemonset/node-exporter", "-n", "observability", "--timeout=120s"},
		)
	}

	for _, readyCheck := range readyChecks {
		kindContainer, err = kindContainer.WithExec(readyCheck).Sync(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to wait for workload to be ready (%s/%s): %w", readyCheck[3], readyCheck[5], err)
		}
	}

	return kindContainer, nil
}

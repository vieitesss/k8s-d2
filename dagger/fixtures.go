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
	includeStorage bool,
) (*dagger.Container, error) {
	var err error

	// Clean up existing namespace to ensure fresh state
	kindContainer, err = kindContainer.
		// Delete namespace if it exists (ignore if not found)
		WithExec([]string{"kubectl", "delete", "namespace", "k8s-d2-test", "--ignore-not-found=true", "--wait=true", "--timeout=60s"}).
		Sync(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to clean up namespace: %w", err)
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

	return kindContainer, nil
}

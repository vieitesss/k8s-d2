package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/vieitesss/k8s-d2/pkg/kube"
	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/render"
)

func runGenerate(cmd *cobra.Command, args []string) error {
	// Configure prettier logging
	log.SetReportTimestamp(false)

	// Suppress k8s/AWS SDK stderr output globally
	oldStderr := os.Stderr
	devNull, devNullErr := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if devNullErr != nil {
		return fmt.Errorf("failed to open /dev/null: %w", devNullErr)
	}
	defer func() {
		os.Stderr = oldStderr
		if devNull != nil {
			devNull.Close()
		}
	}()

	var client *kube.Client
	var cluster *model.Cluster
	var err error

	err = spinner.New().
		Title("Creating K8s client...").
		Action(func() {
			os.Stderr = devNull
			client, err = kube.NewClient(kubeconfig)
			os.Stderr = oldStderr
		}).
		Run()
	if err != nil {
		return err
	}

	opts := kube.FetchOptions{
		Namespace:      namespace,
		AllNamespaces:  allNamespaces,
		IncludeStorage: includeStorage,
	}

	err = spinner.New().
		Title("Fetching cluster topology...").
		Action(func() {
			os.Stderr = devNull
			cluster, err = client.FetchTopology(cmd.Context(), opts)
			os.Stderr = oldStderr
		}).
		Run()
	if err != nil {
		return err
	}

	w := os.Stdout
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	err = spinner.New().
		Title("Rendering D2 diagram...").
		Action(func() {
			renderer := render.NewD2Renderer(w, gridColumns)
			err = renderer.Render(cluster)
		}).
		Run()
	if err != nil {
		return err
	}

	log.Info("D2 diagram generated successfully")
	return nil
}

package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh/spinner"
	"github.com/vieitesss/k8s-d2/pkg/kube"
	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/render"
	"github.com/spf13/cobra"
)

func runGenerate(cmd *cobra.Command, args []string) error {
	var client *kube.Client
	var cluster *model.Cluster
	var err error

	err = spinner.New().
		Title("Creating K8s client...").
		Action(func() {
			client, err = kube.NewClient(kubeconfig)
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
			cluster, err = client.FetchTopology(cmd.Context(), opts)
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

	fmt.Fprintln(os.Stderr, "✓ Done")
	return nil
}

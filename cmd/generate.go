package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/vieitesss/k8s-d2/pkg/kube"
	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/render"
)

func runGenerate(cmd *cobra.Command, args []string) error {
	log.SetReportTimestamp(false)

	client, err := createClientWithSpinner()
	if err != nil {
		return err
	}

	opts := kube.FetchOptions{
		Namespace:      namespace,
		AllNamespaces:  allNamespaces,
		IncludeStorage: includeStorage,
	}

	cluster, err := fetchTopologyWithSpinner(cmd.Context(), client, opts)
	if err != nil {
		return err
	}

	w, closeWriter, err := getOutputWriter()
	if err != nil {
		return err
	}
	defer closeWriter()

	if err := renderWithSpinner(cluster, w); err != nil {
		return err
	}

	log.Info("D2 diagram generated successfully")
	return nil
}

func createClientWithSpinner() (*kube.Client, error) {
	var client *kube.Client
	var clientErr error

	spinnerErr := spinner.New().
		Title("Creating K8s client...").
		Action(func() {
			client, clientErr = kube.NewClient(kubeconfig)
		}).
		Run()

	if spinnerErr != nil {
		return nil, spinnerErr
	}
	return client, clientErr
}

func fetchTopologyWithSpinner(ctx context.Context, client *kube.Client, opts kube.FetchOptions) (*model.Cluster, error) {
	var cluster *model.Cluster
	var fetchErr error

	spinnerErr := spinner.New().
		Title("Fetching cluster topology...").
		Action(func() {
			cluster, fetchErr = client.FetchTopology(ctx, opts)
		}).
		Run()

	if spinnerErr != nil {
		return nil, spinnerErr
	}
	return cluster, fetchErr
}

func getOutputWriter() (*os.File, func(), error) {
	if output == "" {
		return os.Stdout, func() {}, nil
	}

	f, err := os.Create(output)
	if err != nil {
		return nil, nil, err
	}

	return f, func() { f.Close() }, nil
}

func renderWithSpinner(cluster *model.Cluster, w *os.File) error {
	var renderErr error
	spinnerErr := spinner.New().
		Title("Rendering D2 diagram...").
		Action(func() {
			renderer := render.NewD2Renderer(w, gridColumns)
			renderErr = renderer.Render(cluster)
		}).
		Run()

	if spinnerErr != nil {
		return spinnerErr
	}
	return renderErr
}

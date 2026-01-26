package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/vieitesss/k8s-d2/pkg/kroki"
	"github.com/vieitesss/k8s-d2/pkg/kube"
	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/render"
)

func runGenerate(cmd *cobra.Command, args []string) error {
	log.SetReportTimestamp(false)

	// Configure logger for quiet mode - suppress INFO but keep WARN/ERROR
	if rootOptions.quiet {
		log.SetLevel(log.WarnLevel)
	}

	// Validate mutually exclusive flags
	if rootOptions.output != "" && rootOptions.image != "" {
		return errors.New("flags --output/-o and --image/-i are mutually exclusive")
	}

	client, err := createClientWithSpinner()
	if err != nil {
		return err
	}

	opts := kube.FetchOptions{
		Namespace:      rootOptions.namespace,
		AllNamespaces:  rootOptions.allNamespaces,
		IncludeStorage: rootOptions.includeStorage,
	}

	cluster, err := fetchTopologyWithSpinner(cmd.Context(), client, opts)
	if err != nil {
		return err
	}

	// If image output is requested, render to buffer and send to Kroki
	if rootOptions.image != "" {
		return generateImage(cluster)
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

	if rootOptions.quiet {
		client, clientErr = kube.NewClient(rootOptions.kubeconfig)
		return client, clientErr
	}

	spinnerErr := spinner.New().
		Title("Creating K8s client...").
		Action(func() {
			client, clientErr = kube.NewClient(rootOptions.kubeconfig)
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

	if rootOptions.quiet {
		cluster, fetchErr = client.FetchTopology(ctx, opts)
		return cluster, fetchErr
	}

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

// getOutputWriter returns the output file to write the diagram to, a cleanup
// function, and an error. Callers should defer the returned cleanup function
// to ensure any created file is properly closed.
func getOutputWriter() (*os.File, func(), error) {
	if rootOptions.output == "" {
		return os.Stdout, func() {}, nil
	}

	f, err := os.Create(rootOptions.output)
	if err != nil {
		return nil, nil, err
	}

	return f, func() { _ = f.Close() }, nil
}

func renderWithSpinner(cluster *model.Cluster, w *os.File) error {
	var renderErr error

	if rootOptions.quiet {
		renderer := render.NewD2Renderer(w, rootOptions.gridColumns)
		renderErr = renderer.Render(cluster)
		return renderErr
	}

	spinnerErr := spinner.New().
		Title("Rendering D2 diagram...").
		Action(func() {
			renderer := render.NewD2Renderer(w, rootOptions.gridColumns)
			renderErr = renderer.Render(cluster)
		}).
		Run()

	if spinnerErr != nil {
		return spinnerErr
	}
	return renderErr
}

func generateImage(cluster *model.Cluster) error {
	// Ensure output file has .svg extension (Kroki only supports SVG for D2)
	outputFile := rootOptions.image
	if strings.ToLower(filepath.Ext(outputFile)) != ".svg" {
		outputFile = strings.TrimSuffix(outputFile, filepath.Ext(outputFile)) + ".svg"
	}

	// Render D2 to buffer
	var buf bytes.Buffer
	var renderErr error

	if rootOptions.quiet {
		renderer := render.NewD2Renderer(&buf, rootOptions.gridColumns)
		renderErr = renderer.Render(cluster)
	} else {
		spinnerErr := spinner.New().
			Title("Rendering D2 diagram...").
			Action(func() {
				renderer := render.NewD2Renderer(&buf, rootOptions.gridColumns)
				renderErr = renderer.Render(cluster)
			}).
			Run()
		if spinnerErr != nil {
			return spinnerErr
		}
	}
	if renderErr != nil {
		return renderErr
	}

	// Send to Kroki
	var svgData []byte
	var krokiErr error
	krokiClient := kroki.NewClient()

	if rootOptions.quiet {
		svgData, krokiErr = krokiClient.GenerateSVG(buf.String())
	} else {
		spinnerErr := spinner.New().
			Title("Generating SVG via Kroki...").
			Action(func() {
				svgData, krokiErr = krokiClient.GenerateSVG(buf.String())
			}).
			Run()
		if spinnerErr != nil {
			return spinnerErr
		}
	}
	if krokiErr != nil {
		return krokiErr
	}

	// Write SVG to file
	if err := os.WriteFile(outputFile, svgData, 0644); err != nil {
		return err
	}

	log.Info("SVG image generated successfully", "file", outputFile)
	return nil
}

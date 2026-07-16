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

// runWithSpinner executes an action with a spinner in normal mode,
// or directly in quiet mode.
func runWithSpinner(title string, action func() error) error {
	if rootOptions.quiet {
		return action()
	}
	var actionErr error
	if err := spinner.New().Title(title).Action(func() {
		actionErr = action()
	}).Run(); err != nil {
		return err
	}
	return actionErr
}

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

	if err := validateImageOptions(); err != nil {
		return err
	}

	client, err := createClientWithSpinner()
	if err != nil {
		return err
	}

	opts := kube.FetchOptions{
		Namespaces:     rootOptions.namespaces,
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

func validateImageOptions() error {
	if rootOptions.image == "" {
		return nil
	}

	rootOptions.krokiBaseURL = strings.TrimSpace(rootOptions.krokiBaseURL)
	if rootOptions.krokiBaseURL == "" {
		return errors.New("flag --kroki-base-url cannot be empty when using --image")
	}

	if rootOptions.krokiTimeout <= 0 {
		return errors.New("flag --kroki-timeout must be greater than 0 when using --image")
	}

	return nil
}

func createClientWithSpinner() (*kube.Client, error) {
	var client *kube.Client
	err := runWithSpinner("Creating K8s client...", func() error {
		var clientErr error
		client, clientErr = kube.NewClient(rootOptions.kubeconfig)
		return clientErr
	})
	return client, err
}

func fetchTopologyWithSpinner(ctx context.Context, client *kube.Client, opts kube.FetchOptions) (*model.Cluster, error) {
	var cluster *model.Cluster
	err := runWithSpinner("Fetching cluster topology...", func() error {
		var fetchErr error
		cluster, fetchErr = client.FetchTopology(ctx, opts)
		return fetchErr
	})
	return cluster, err
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
	return runWithSpinner("Rendering D2 diagram...", func() error {
		renderer := render.NewD2Renderer(w)
		return renderer.Render(cluster)
	})
}

func generateImage(cluster *model.Cluster) error {
	// Ensure output file has .svg extension (Kroki only supports SVG for D2)
	outputFile := rootOptions.image
	if strings.ToLower(filepath.Ext(outputFile)) != ".svg" {
		outputFile = strings.TrimSuffix(outputFile, filepath.Ext(outputFile)) + ".svg"
	}

	// Render D2 to buffer
	var buf bytes.Buffer

	if err := runWithSpinner("Rendering D2 diagram...", func() error {
		renderer := render.NewD2Renderer(&buf)
		return renderer.Render(cluster)
	}); err != nil {
		return err
	}

	// Send to Kroki
	var svgData []byte
	krokiClient := kroki.NewClientWithOptions(kroki.Options{
		BaseURL: rootOptions.krokiBaseURL,
		Timeout: rootOptions.krokiTimeout,
	})

	if err := runWithSpinner("Generating SVG via Kroki...", func() error {
		var krokiErr error
		svgData, krokiErr = krokiClient.GenerateSVG(buf.String())
		return krokiErr
	}); err != nil {
		return err
	}

	// Write SVG to file
	if err := os.WriteFile(outputFile, svgData, 0644); err != nil {
		return err
	}

	log.Info("SVG image generated successfully", "file", outputFile)
	return nil
}

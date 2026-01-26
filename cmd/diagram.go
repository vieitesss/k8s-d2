package cmd

import (
	"github.com/spf13/cobra"
)

var diagramCmd = &cobra.Command{
	Use:   "diagram",
	Short: "Generate D2 diagrams from Kubernetes cluster topology",
	Long: `Generate D2 diagram files visualizing namespaces, workloads,
services, and their relationships from your Kubernetes cluster.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runGenerate,
}

func init() {
	rootCmd.AddCommand(diagramCmd)

	diagramCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(err)
		cmd.Println()
		cmd.Help()
		return err
	})

	diagramCmd.Flags().StringVar(&rootOptions.kubeconfig, "kubeconfig", "", "path to kubeconfig (default: ~/.kube/config)")
	diagramCmd.Flags().StringVarP(&rootOptions.namespace, "namespace", "n", "", "namespace to visualize (default: all non-system)")
	diagramCmd.Flags().BoolVarP(&rootOptions.allNamespaces, "all-namespaces", "A", false, "include all namespaces (including system)")
	diagramCmd.Flags().StringVarP(&rootOptions.output, "output", "o", "", "output D2 file (default: stdout)")
	diagramCmd.Flags().StringVarP(&rootOptions.image, "image", "i", "", "output .svg image file. Extension is not needed always SVG file is generated")
	diagramCmd.Flags().BoolVar(&rootOptions.includeStorage, "include-storage", false, "include PVC/StorageClass layer")
	diagramCmd.Flags().IntVar(&rootOptions.gridColumns, "grid-columns", 3, "number of columns in grid layout (0 for single column)")
	diagramCmd.Flags().BoolVarP(&rootOptions.quiet, "quiet", "q", false, "suppress progress indicators and log messages")
}

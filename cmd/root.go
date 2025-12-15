package cmd

import (
	"github.com/spf13/cobra"
)

var (
	kubeconfig     string
	namespace      string
	allNamespaces  bool
	output         string
	includeStorage bool
	gridColumns    int
	showVersion    bool
)

var rootCmd = &cobra.Command{
	Use:   "k8sdd",
	Short: "Generate D2 diagrams from Kubernetes cluster topology",
	Long: `k8s-d2 queries your Kubernetes cluster and generates D2 diagram files
visualizing namespaces, workloads, services, and their relationships.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runGenerate,
}

func init() {
	rootCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig (default: ~/.kube/config)")
	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace to visualize (default: all non-system)")
	rootCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "include all namespaces (including system)")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "output file (default: stdout)")
	rootCmd.Flags().BoolVar(&includeStorage, "include-storage", false, "include PVC/StorageClass layer")
	rootCmd.Flags().IntVar(&gridColumns, "grid-columns", 3, "number of columns in grid layout (0 for single column)")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show version information")
}

func Execute(version string) error {
	rootCmd.Version = version
	return rootCmd.Execute()
}

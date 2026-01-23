package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

type RootOptions struct {
	kubeconfig     string
	namespace      string
	allNamespaces  bool
	output         string
	image          string
	includeStorage bool
	gridColumns    int
	showVersion    bool
	quiet          bool
}

var rootOptions RootOptions

var rootCmd = &cobra.Command{
	Use:   "k8sdd",
	Short: "Generate D2 diagrams from Kubernetes cluster topology",
	Long: `k8s-d2 queries your Kubernetes cluster and generates D2 diagram files
visualizing namespaces, workloads, services, and their relationships.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runRoot,
}

func runRoot(cmd *cobra.Command, args []string) error {
	log.Warn("DEPRECATED: Running k8sdd without a subcommand is deprecated. Please use 'k8sdd diagram' instead. This will be removed in v1.0.0.")
	return runGenerate(cmd, args)
}

func init() {
	rootCmd.Flags().StringVar(&rootOptions.kubeconfig, "kubeconfig", "", "path to kubeconfig (default: ~/.kube/config)")
	rootCmd.Flags().StringVarP(&rootOptions.namespace, "namespace", "n", "", "namespace to visualize (default: all non-system)")
	rootCmd.Flags().BoolVarP(&rootOptions.allNamespaces, "all-namespaces", "A", false, "include all namespaces (including system)")
	rootCmd.Flags().StringVarP(&rootOptions.output, "output", "o", "", "output file (default: stdout)")
	rootCmd.Flags().BoolVar(&rootOptions.includeStorage, "include-storage", false, "include PVC/StorageClass layer")
	rootCmd.Flags().IntVar(&rootOptions.gridColumns, "grid-columns", 3, "number of columns in grid layout (0 for single column)")
	rootCmd.Flags().BoolVarP(&rootOptions.showVersion, "version", "v", false, "show version information")
	rootCmd.Flags().BoolVarP(&rootOptions.quiet, "quiet", "q", false, "suppress progress indicators and log messages")
}

func Execute(version string) error {
	rootCmd.Version = version
	return rootCmd.Execute()
}

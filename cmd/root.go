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
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(err)
		cmd.Println()
		_ = cmd.Help()
		return err
	})

	rootCmd.PersistentFlags().StringVar(&rootOptions.kubeconfig, "kubeconfig", "", "path to kubeconfig (default: ~/.kube/config)")
	rootCmd.PersistentFlags().StringVarP(&rootOptions.namespace, "namespace", "n", "", "namespace to visualize (default: all non-system)")
	rootCmd.PersistentFlags().BoolVarP(&rootOptions.allNamespaces, "all-namespaces", "A", false, "include all namespaces (including system)")
	rootCmd.PersistentFlags().StringVarP(&rootOptions.output, "output", "o", "", "output D2 file (default: stdout)")
	rootCmd.PersistentFlags().StringVarP(&rootOptions.image, "image", "i", "", "output .svg image file. Extension is not needed always SVG file is generated")
	rootCmd.PersistentFlags().BoolVar(&rootOptions.includeStorage, "include-storage", false, "include PVC/StorageClass layer")
	rootCmd.PersistentFlags().IntVar(&rootOptions.gridColumns, "grid-columns", 3, "deprecated: automatic layout is used")
	if err := rootCmd.PersistentFlags().MarkDeprecated("grid-columns", "automatic layout is now used; this flag has no effect"); err != nil {
		panic(err)
	}
	rootCmd.Flags().BoolVarP(&rootOptions.showVersion, "version", "v", false, "show version information")
	rootCmd.PersistentFlags().BoolVarP(&rootOptions.quiet, "quiet", "q", false, "suppress progress indicators and log messages")
}

func Execute(version string) error {
	rootCmd.Version = version
	return rootCmd.Execute()
}

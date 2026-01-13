package cmd

import (
	"github.com/spf13/cobra"
)

var diagramCmd = &cobra.Command{
	Use:   "diagram",
	Short: "Generate D2 diagram from Kubernetes cluster topology",
	Long:  `Generate a D2 diagram file visualizing namespaces, workloads, services, and their relationships from your Kubernetes cluster.`,
	RunE:  runGenerate,
}

func init() {
	rootCmd.AddCommand(diagramCmd)
}

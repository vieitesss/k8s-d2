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
		_ = cmd.Help()
		return err
	})
}

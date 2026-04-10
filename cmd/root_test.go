package cmd

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestRootAndDiagramExposeSameGenerationFlags(t *testing.T) {
	tests := []struct {
		name      string
		shorthand string
	}{
		{name: "kubeconfig"},
		{name: "namespace", shorthand: "n"},
		{name: "all-namespaces", shorthand: "A"},
		{name: "output", shorthand: "o"},
		{name: "image", shorthand: "i"},
		{name: "include-storage"},
		{name: "grid-columns"},
		{name: "quiet", shorthand: "q"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootFlag := rootCmd.Flag(tt.name)
			if rootFlag == nil {
				t.Fatalf("expected root command to expose %q", tt.name)
			}

			diagramFlag := diagramCmd.Flag(tt.name)
			if diagramFlag == nil {
				t.Fatalf("expected diagram command to expose %q", tt.name)
			}

			if rootFlag.Shorthand != tt.shorthand {
				t.Fatalf("expected root command %q shorthand %q, got %q", tt.name, tt.shorthand, rootFlag.Shorthand)
			}

			if diagramFlag.Shorthand != tt.shorthand {
				t.Fatalf("expected diagram command %q shorthand %q, got %q", tt.name, tt.shorthand, diagramFlag.Shorthand)
			}
		})
	}
}

func TestGridColumnsFlagIsDeprecated(t *testing.T) {
	tests := []struct {
		name string
		flag string
	}{
		{name: "root", flag: rootCmd.Flag("grid-columns").Deprecated},
		{name: "diagram", flag: diagramCmd.Flag("grid-columns").Deprecated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.flag == "" {
				t.Fatalf("expected grid-columns flag to be deprecated")
			}
		})
	}
}

func TestGridColumnsFlagIsStillAccepted(t *testing.T) {
	tests := []struct {
		name string
		cmd  interface{ ParseFlags([]string) error }
		args []string
	}{
		{name: "root", cmd: rootCmd, args: []string{"--grid-columns", "4"}},
		{name: "diagram", cmd: diagramCmd, args: []string{"--grid-columns", "5"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootOptions.gridColumns = 3
			if err := tt.cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("expected deprecated grid-columns flag to remain accepted: %v", err)
			}
			if rootOptions.gridColumns == 3 {
				t.Fatalf("expected deprecated grid-columns flag to still parse a value")
			}
		})
	}
}

func TestImageFlagIsAcceptedOnRootAndDiagram(t *testing.T) {
	tests := []struct {
		name string
		cmd  interface{ ParseFlags([]string) error }
		args []string
		want string
	}{
		{name: "root", cmd: rootCmd, args: []string{"-i", "root.svg"}, want: "root.svg"},
		{name: "diagram", cmd: diagramCmd, args: []string{"-i", "diagram.svg"}, want: "diagram.svg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootOptions.image = ""
			if err := tt.cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("expected image flag to parse: %v", err)
			}
			if rootOptions.image != tt.want {
				t.Fatalf("expected image flag to set %q, got %q", tt.want, rootOptions.image)
			}
		})
	}
}

func TestNamespaceFlagSupportsMultipleValues(t *testing.T) {
	tests := []struct {
		name string
		cmd  interface{ Flags() *pflag.FlagSet }
	}{
		{name: "root", cmd: rootCmd},
		{name: "diagram", cmd: diagramCmd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := tt.cmd.Flags().Lookup("namespace")
			if flag == nil {
				t.Fatalf("expected namespace flag to be registered")
			}
			if got := flag.Value.Type(); got != "stringSlice" {
				t.Fatalf("expected namespace flag type stringSlice, got %q", got)
			}
		})
	}
}

package cmd

import "testing"

func TestGridColumnsFlagIsDeprecated(t *testing.T) {
	tests := []struct {
		name string
		flag string
	}{
		{name: "root", flag: rootCmd.Flags().Lookup("grid-columns").Deprecated},
		{name: "diagram", flag: diagramCmd.Flags().Lookup("grid-columns").Deprecated},
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

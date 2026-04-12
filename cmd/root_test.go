package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/pflag"
	"github.com/vieitesss/k8s-d2/internal/validation"
	"github.com/vieitesss/k8s-d2/pkg/kroki"
	"github.com/vieitesss/k8s-d2/pkg/render"
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
		{name: "kroki-base-url"},
		{name: "kroki-timeout"},
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

func TestIncludeStorageFlagDescriptionMatchesCurrentBehavior(t *testing.T) {
	for _, cmd := range []struct {
		name string
		flag *pflag.Flag
	}{
		{name: "root", flag: rootCmd.Flag("include-storage")},
		{name: "diagram", flag: diagramCmd.Flag("include-storage")},
	} {
		t.Run(cmd.name, func(t *testing.T) {
			if cmd.flag == nil {
				t.Fatalf("expected include-storage flag to be registered")
			}
			if cmd.flag.Usage != includeStorageUsage {
				t.Fatalf("expected include-storage usage %q, got %q", includeStorageUsage, cmd.flag.Usage)
			}
		})
	}
}

func TestIncludeStorageFlagKeepsPVCOnlyRenderContract(t *testing.T) {
	fixtureData := loadFixtureFiles(t,
		filepath.Join("..", "test", "fixtures", "base", "04-statefulsets.yaml"),
		filepath.Join("..", "test", "fixtures", "storage", "01-storageclass.yaml"),
		filepath.Join("..", "test", "fixtures", "storage", "02-pvcs.yaml"),
	)

	cluster, err := validation.NewFixtureParser("k8s-d2-test", true).ParseFixtures(fixtureData)
	if err != nil {
		t.Fatalf("failed to parse storage fixtures: %v", err)
	}

	var buf bytes.Buffer
	renderer := render.NewD2Renderer(&buf, 0)
	if err := renderer.Render(cluster); err != nil {
		t.Fatalf("failed to render storage fixtures: %v", err)
	}

	output := buf.String()
	for _, pvcName := range []string{"logs-volume", "data-database-0", "data-database-1"} {
		if !strings.Contains(output, render.PVCID(pvcName)+": {") {
			t.Fatalf("expected PVC %q to render in storage output\n%s", pvcName, output)
		}
	}

	if got := strings.Count(output, "[standard]"); got != 3 {
		t.Fatalf("expected 3 PVC storage class labels in storage output, got %d\n%s", got, output)
	}
	if strings.Contains(output, render.SanitizeID("standard")+": {") {
		t.Fatalf("did not expect storage class %q to render as a standalone node\n%s", "standard", output)
	}
	if strings.Contains(output, "label: \"standard\"") {
		t.Fatalf("did not expect storage class %q to render as a standalone label\n%s", "standard", output)
	}
	if strings.Contains(output, "StorageClass") {
		t.Fatalf("did not expect StorageClass resources to render as nodes\n%s", output)
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

func TestKrokiFlagsAreAcceptedOnRootAndDiagram(t *testing.T) {
	tests := []struct {
		name        string
		cmd         interface{ ParseFlags([]string) error }
		args        []string
		wantBaseURL string
		wantTimeout time.Duration
	}{
		{
			name:        "root",
			cmd:         rootCmd,
			args:        []string{"--kroki-base-url", "https://kroki.internal", "--kroki-timeout", "45s"},
			wantBaseURL: "https://kroki.internal",
			wantTimeout: 45 * time.Second,
		},
		{
			name:        "diagram",
			cmd:         diagramCmd,
			args:        []string{"--kroki-base-url", "https://kroki.example.com/api", "--kroki-timeout", "2m"},
			wantBaseURL: "https://kroki.example.com/api",
			wantTimeout: 2 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootOptions.krokiBaseURL = kroki.DefaultBaseURL
			rootOptions.krokiTimeout = kroki.DefaultTimeout

			if err := tt.cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("expected Kroki flags to parse: %v", err)
			}

			if rootOptions.krokiBaseURL != tt.wantBaseURL {
				t.Fatalf("expected Kroki base URL %q, got %q", tt.wantBaseURL, rootOptions.krokiBaseURL)
			}

			if rootOptions.krokiTimeout != tt.wantTimeout {
				t.Fatalf("expected Kroki timeout %s, got %s", tt.wantTimeout, rootOptions.krokiTimeout)
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

func loadFixtureFiles(t *testing.T, paths ...string) [][]byte {
	t.Helper()

	fixtureData := make([][]byte, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read fixture %q: %v", path, err)
		}
		fixtureData = append(fixtureData, data)
	}

	return fixtureData
}

func TestRunRootQuietSuppressesDeprecatedWarning(t *testing.T) {
	originalOptions := rootOptions
	rootOptions = RootOptions{
		kubeconfig: "/definitely/missing-kubeconfig",
		quiet:      true,
	}
	t.Cleanup(func() {
		rootOptions = originalOptions
	})

	var logOutput bytes.Buffer
	defaultLogger := log.Default()
	originalLevel := defaultLogger.GetLevel()
	log.SetOutput(&logOutput)
	log.SetLevel(log.InfoLevel)
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetLevel(originalLevel)
	})

	err := runRoot(rootCmd, nil)
	if err == nil {
		t.Fatal("expected runRoot to fail with invalid kubeconfig")
	}

	output := logOutput.String()
	if strings.Contains(output, "DEPRECATED: Running k8sdd without a subcommand is deprecated") {
		t.Fatalf("expected quiet root path to suppress deprecated warning, got %q", output)
	}
}

func TestRunRootWarnsWhenNotQuiet(t *testing.T) {
	originalOptions := rootOptions
	rootOptions = RootOptions{
		kubeconfig: "/definitely/missing-kubeconfig",
	}
	t.Cleanup(func() {
		rootOptions = originalOptions
	})

	var logOutput bytes.Buffer
	defaultLogger := log.Default()
	originalLevel := defaultLogger.GetLevel()
	log.SetOutput(&logOutput)
	log.SetLevel(log.InfoLevel)
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetLevel(originalLevel)
	})

	err := runRoot(rootCmd, nil)
	if err == nil {
		t.Fatal("expected runRoot to fail with invalid kubeconfig")
	}

	output := logOutput.String()
	if !strings.Contains(output, "DEPRECATED: Running k8sdd without a subcommand is deprecated") {
		t.Fatalf("expected non-quiet root path to emit deprecated warning, got %q", output)
	}
}

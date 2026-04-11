package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vieitesss/k8s-d2/pkg/model"
)

func TestValidateImageOptions(t *testing.T) {
	tests := []struct {
		name    string
		options RootOptions
		wantErr string
	}{
		{
			name: "non-image output skips Kroki validation",
			options: RootOptions{
				output:       "cluster.d2",
				krokiTimeout: -1 * time.Second,
			},
		},
		{
			name: "blank base URL is rejected for image generation",
			options: RootOptions{
				image:        "cluster.svg",
				krokiBaseURL: "   ",
				krokiTimeout: time.Second,
			},
			wantErr: "flag --kroki-base-url cannot be empty when using --image",
		},
		{
			name: "non-positive timeout is rejected for image generation",
			options: RootOptions{
				image:        "cluster.svg",
				krokiBaseURL: "https://kroki.internal",
				krokiTimeout: 0,
			},
			wantErr: "flag --kroki-timeout must be greater than 0 when using --image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalOptions := rootOptions
			rootOptions = tt.options
			t.Cleanup(func() {
				rootOptions = originalOptions
			})

			err := validateImageOptions()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}

			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestGenerateImageUsesConfiguredKrokiBaseURL(t *testing.T) {
	requestedPath := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "text/plain" {
			t.Errorf("expected Content-Type text/plain, got %s", ct)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "cluster")

	originalOptions := rootOptions
	t.Cleanup(func() {
		rootOptions = originalOptions
	})

	rootOptions.image = outputFile
	rootOptions.krokiBaseURL = server.URL + "/"
	rootOptions.krokiTimeout = time.Second
	rootOptions.quiet = true

	cluster := &model.Cluster{
		Name: "test-cluster",
		Namespaces: []model.Namespace{{
			Name: "default",
		}},
	}

	if err := generateImage(cluster); err != nil {
		t.Fatalf("generateImage returned error: %v", err)
	}

	if requestedPath != "/d2/svg" {
		t.Fatalf("expected request path /d2/svg, got %q", requestedPath)
	}

	svgPath := outputFile + ".svg"
	content, err := os.ReadFile(svgPath)
	if err != nil {
		t.Fatalf("expected SVG file %q to be written: %v", svgPath, err)
	}

	if !strings.Contains(string(content), "<svg") {
		t.Fatalf("expected written file to contain SVG markup, got %q", string(content))
	}
}

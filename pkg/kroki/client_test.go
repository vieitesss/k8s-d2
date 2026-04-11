package kroki

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	client := NewClient()

	if client.baseURL != DefaultBaseURL {
		t.Fatalf("expected default base URL %q, got %q", DefaultBaseURL, client.baseURL)
	}

	if client.httpClient.Timeout != DefaultTimeout {
		t.Fatalf("expected default timeout %s, got %s", DefaultTimeout, client.httpClient.Timeout)
	}
}

func TestNewClientUsesConfiguredBaseURLAndTimeout(t *testing.T) {
	client := NewClientWithOptions(Options{
		BaseURL: "https://kroki.internal/",
		Timeout: 45 * time.Second,
	})

	if client.baseURL != "https://kroki.internal" {
		t.Fatalf("expected normalized base URL %q, got %q", "https://kroki.internal", client.baseURL)
	}

	if client.httpClient.Timeout != 45*time.Second {
		t.Fatalf("expected timeout %s, got %s", 45*time.Second, client.httpClient.Timeout)
	}
}

func TestGenerateSVG_Success(t *testing.T) {
	mockSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect/></svg>`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/d2/svg" {
			t.Errorf("expected path /d2/svg, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("expected Content-Type text/plain, got %s", r.Header.Get("Content-Type"))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mockSVG)
	}))
	defer server.Close()

	client := NewClientWithOptions(Options{BaseURL: server.URL + "/", Timeout: time.Second})

	result, err := client.GenerateSVG("a -> b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) == 0 {
		t.Error("expected non-empty SVG data")
	}

	if len(result) != len(mockSVG) {
		t.Errorf("expected %d bytes, got %d", len(mockSVG), len(result))
	}
}

func TestGenerateSVG_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid diagram syntax"))
	}))
	defer server.Close()

	client := NewClientWithOptions(Options{BaseURL: server.URL, Timeout: time.Second})

	_, err := client.GenerateSVG("invalid diagram")
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestGenerateSVG_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := NewClientWithOptions(Options{BaseURL: server.URL, Timeout: time.Second})

	_, err := client.GenerateSVG("a -> b")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

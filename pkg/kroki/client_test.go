package kroki

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
		w.Write(mockSVG)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

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
		w.Write([]byte("invalid diagram syntax"))
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	_, err := client.GenerateSVG("invalid diagram")
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestGenerateSVG_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	_, err := client.GenerateSVG("a -> b")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

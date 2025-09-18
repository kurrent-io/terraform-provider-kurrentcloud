package client

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func TestUserAgentTransport(t *testing.T) {
	// Set a test version
	originalVersion := Version
	Version = "1.2.3"
	defer func() { Version = originalVersion }()

	// Create a test server that captures the User-Agent header
	var capturedUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a client with our User-Agent transport
	client := newHTTPClientWithUserAgent(http.DefaultClient)

	// Make a request
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	// Verify the User-Agent header was set correctly
	expectedPrefix := "terraform-provider-kurrentcloud/1.2.3"
	if !strings.HasPrefix(capturedUserAgent, expectedPrefix) {
		t.Errorf("Expected User-Agent to start with %q, got %q", expectedPrefix, capturedUserAgent)
	}

	// Verify it contains Go version and OS info
	if !strings.Contains(capturedUserAgent, runtime.Version()) {
		t.Errorf(
			"Expected User-Agent to contain Go version %q, got %q",
			runtime.Version(),
			capturedUserAgent,
		)
	}

	if !strings.Contains(capturedUserAgent, runtime.GOOS) {
		t.Errorf("Expected User-Agent to contain OS %q, got %q", runtime.GOOS, capturedUserAgent)
	}

	if !strings.Contains(capturedUserAgent, runtime.GOARCH) {
		t.Errorf(
			"Expected User-Agent to contain architecture %q, got %q",
			runtime.GOARCH,
			capturedUserAgent,
		)
	}
}

func TestUserAgentTransportPreservesExisting(t *testing.T) {
	// Create a test server that captures the User-Agent header
	var capturedUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a client with our User-Agent transport
	client := newHTTPClientWithUserAgent(http.DefaultClient)

	// Make a request with an existing User-Agent
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	customUserAgent := "custom-user-agent/1.0"
	req.Header.Set("User-Agent", customUserAgent)

	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	// Verify the existing User-Agent header was preserved
	if capturedUserAgent != customUserAgent {
		t.Errorf(
			"Expected User-Agent to be preserved as %q, got %q",
			customUserAgent,
			capturedUserAgent,
		)
	}
}

func TestBuildUserAgent(t *testing.T) {
	// Set a test version
	originalVersion := Version
	originalTerraformVersion := TerraformVersion
	Version = "1.2.3"
	TerraformVersion = "1.6.0"
	defer func() {
		Version = originalVersion
		TerraformVersion = originalTerraformVersion
	}()

	userAgent := BuildUserAgent()

	expected := "terraform-provider-kurrentcloud/1.2.3 Terraform/1.6.0"
	if !strings.HasPrefix(userAgent, expected) {
		t.Errorf("Expected User-Agent to start with %q, got %q", expected, userAgent)
	}

	// Verify it contains all expected components
	expectedComponents := []string{
		"terraform-provider-kurrentcloud/1.2.3",
		"Terraform/1.6.0",
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	}

	for _, component := range expectedComponents {
		if !strings.Contains(userAgent, component) {
			t.Errorf("Expected User-Agent to contain %q, got %q", component, userAgent)
		}
	}
}

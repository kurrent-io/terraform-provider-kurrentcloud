package client

import (
	"fmt"
	"net/http"
	"runtime"
)

var (
	// Version will be set at build time via ldflags
	Version = "dev"
	// TerraformVersion can be detected from environment or context
	TerraformVersion = "unknown"
)

// userAgentTransport is a custom RoundTripper that adds User-Agent headers
type userAgentTransport struct {
	transport http.RoundTripper
	userAgent string
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Only set User-Agent if not already present
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", t.userAgent)
	}
	return t.transport.RoundTrip(req)
}

// BuildUserAgent constructs the User-Agent string following Terraform provider conventions
func BuildUserAgent() string {
	return fmt.Sprintf("terraform-provider-kurrentcloud/%s Terraform/%s Go/%s (%s/%s)",
		Version,
		TerraformVersion,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// newHTTPClientWithUserAgent creates an HTTP client with User-Agent telemetry
func newHTTPClientWithUserAgent(baseClient *http.Client) *http.Client {
	if baseClient == nil {
		baseClient = http.DefaultClient
	}

	transport := baseClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	return &http.Client{
		Transport: &userAgentTransport{
			transport: transport,
			userAgent: BuildUserAgent(),
		},
		CheckRedirect: baseClient.CheckRedirect,
		Jar:           baseClient.Jar,
		Timeout:       baseClient.Timeout,
	}
}

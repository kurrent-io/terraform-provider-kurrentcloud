package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/go-cleanhttp"
)

type Config struct {
	URL                 string
	IdentityProviderURL string
	IdentityKitURL      string
	ClientID            string
	ClientSecret        string
	TokenStore          string
	RefreshToken        string
}

// serviceAccountMode reports whether the config selects Service Account
// (client-credentials) authentication, i.e. a client secret is present.
func (config *Config) serviceAccountMode() bool {
	return strings.TrimSpace(config.ClientSecret) != ""
}

func (config *Config) validate() error {
	if strings.TrimSpace(config.URL) == "" {
		return errors.New("URL is required")
	}

	// Service Account authentication is memory-only: it never reads or writes
	// the on-disk token store, so don't require (or create) it. This keeps SA
	// mode usable in environments with no writable filesystem.
	if config.serviceAccountMode() {
		return nil
	}

	if _, err := os.Stat(config.TokenStore); err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(config.TokenStore, 0o700)
			if err != nil {
				return fmt.Errorf("cannot create path %q: %w", config.TokenStore, err)
			}

			return nil
		}

		return fmt.Errorf("error reading Token Store %q: %w", config.TokenStore, err)
	}

	return nil
}

type Client struct {
	apiURL *url.URL

	audience     string
	idpURL       *url.URL
	idpKitURL    *url.URL
	tokenStore   *tokenStore
	clientID     string
	clientSecret string
	refreshToken string

	// saTokenMu guards saToken. Service Account tokens are cached in memory
	// only and are never written to the on-disk token store.
	saTokenMu sync.Mutex
	saToken   *tokenData

	httpClient *http.Client
}

func New(opts *Config) (*Client, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	// Service Account authentication requires both halves of the credential
	// pair; a secret without an explicit client id would otherwise silently
	// fall back to the default interactive client id.
	if strings.TrimSpace(opts.ClientSecret) != "" && strings.TrimSpace(opts.ClientID) == "" {
		return nil, errors.New("client_id and client_secret must both be set for service account authentication")
	}

	tokenStore := &tokenStore{
		path: opts.TokenStore,
	}

	apiURL, err := url.Parse(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid service URL %q: %w", opts.URL, err)
	}

	identityProviderURL := opts.IdentityProviderURL
	if strings.TrimSpace(identityProviderURL) == "" {
		identityProviderURL = "https://identity.eventstore.com"
	}
	parsedIdentityProviderURL, err := url.Parse(identityProviderURL)
	if err != nil {
		return nil, fmt.Errorf("invalid identity provider URL: %q, %w", identityProviderURL, err)
	}

	identityKitURL := opts.IdentityKitURL
	if strings.TrimSpace(identityKitURL) == "" {
		identityKitURL = "https://thorough-shelter-35.authkit.app"
	}
	parsedIdentityKitURL, err := url.Parse(identityKitURL)
	if err != nil {
		return nil, fmt.Errorf("invalid identity kit URL: %q, %w", identityKitURL, err)
	}

	clientID := opts.ClientID
	if strings.TrimSpace(clientID) == "" {
		clientID = "OraYp3cFES9O8aWuQtnqi1A7m534iTwt"
	}

	return &Client{
		apiURL:       apiURL,
		audience:     "api.eventstore.cloud",
		idpURL:       parsedIdentityProviderURL,
		idpKitURL:    parsedIdentityKitURL,
		clientID:     clientID,
		clientSecret: strings.TrimSpace(opts.ClientSecret),
		tokenStore:   tokenStore,
		refreshToken: opts.RefreshToken,
		httpClient:   newHTTPClientWithUserAgent(cleanhttp.DefaultClient()),
	}, nil
}

func (c *Client) addAuthorizationHeader(req *http.Request) diag.Diagnostics {
	token, err := c.accessToken(false)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error obtaining access token: %w", err))
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	return nil
}

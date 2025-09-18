package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/go-cleanhttp"
)

type Config struct {
	URL                 string
	IdentityProviderURL string
	ClientID            string
	TokenStore          string
	RefreshToken        string
}

func (config *Config) validate() error {
	if strings.TrimSpace(config.URL) == "" {
		return errors.New("URL is required")
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
	tokenStore   *tokenStore
	clientID     string
	refreshToken string

	httpClient *http.Client
}

func New(opts *Config) (*Client, error) {
	if err := opts.validate(); err != nil {
		return nil, err
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

	clientID := opts.ClientID
	if strings.TrimSpace(clientID) == "" {
		clientID = "OraYp3cFES9O8aWuQtnqi1A7m534iTwt"
	}

	return &Client{
		apiURL:       apiURL,
		audience:     "api.eventstore.cloud",
		idpURL:       parsedIdentityProviderURL,
		clientID:     clientID,
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

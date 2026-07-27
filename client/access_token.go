package client

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwt"
)

type accessToken string

func (a accessToken) IsValid() (jwt.Token, bool) {
	block, _ := pem.Decode([]byte(jwtPublicKey))
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, false
	}
	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)

	token, err := jwt.ParseString(string(a),
		jwt.WithVerify(jwa.RS256, rsaPublicKey))
	// Invalid signature
	if err != nil {
		return token, false
	}

	err = jwt.Validate(token, jwt.WithAcceptableSkew(30*time.Second))
	// Token has expired
	if err != nil {
		return token, false
	}

	for _, tokenAud := range token.Audience() {
		for _, aud := range []string{"https://api.eventstore.cloud", "qB1dK9gAx6U1H1miH4LfwCp4Q1y3qSeZ"} {
			if aud == tokenAud {
				return token, true
			}
		}
	}

	return token, true
}

// Inspect a token from the local token store
func (c *Client) TokenInspect(audience string) (*tokenData, error) {
	return c.tokenStore.get(audience)
}

// Convenience function
func (c *Client) TokenRefresh(force bool) error {
	_, err := c.accessToken(force)

	return err
}

// closeIgnoreError closes closer, discarding any error. It is meant for
// direct use in defer statements: `defer closeIgnoreError(resp.Body)`.
func closeIgnoreError(closer io.Closer) {
	_ = closer.Close()
}

// isUsable reports whether the token parses as a JWT and has not expired
// (with 30s of acceptable skew). The signature is deliberately not verified:
// the provider is the token bearer, not a resource server, and only reads the
// payload to decide whether to mint a new token. Service Account tokens are
// not signed with the embedded key, so signature verification is not possible
// for them anyway.
func (a accessToken) isUsable() bool {
	token, err := jwt.ParseString(string(a))
	if err != nil {
		return false
	}

	return jwt.Validate(token, jwt.WithAcceptableSkew(30*time.Second)) == nil
}

// serviceAccountToken mints an access token via the OAuth2 client-credentials
// grant for Service Account (machine-to-machine) authentication against the
// identity kit token endpoint. Only grant_type/client_id/client_secret are
// sent (the endpoint rejects audience/scope on this grant) and the returned
// token has no refresh token. Tokens are cached in memory only — Service
// Account credentials and tokens are never written to the on-disk token store.
func (c *Client) serviceAccountToken(force bool) (*tokenData, error) {
	c.saTokenMu.Lock()
	defer c.saTokenMu.Unlock()

	if c.saToken != nil && !force && c.saToken.AccessToken.isUsable() {
		return c.saToken, nil
	}

	log.Printf("[INFO] authenticating via service account client credentials (client_id=%s)", c.clientID)

	idpKitURL := *c.idpKitURL
	idpKitURL.Path = "/oauth2/token"

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)

	resp, err := c.httpClient.PostForm(idpKitURL.String(), form)
	if err != nil {
		return nil, fmt.Errorf("error requesting service account access token: %w", err)
	}
	defer closeIgnoreError(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error %d requesting service account access token", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	result := tokenData{}
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing IDP response: %w", err)
	}

	c.saToken = &result

	return &result, nil
}

func (c *Client) accessToken(force bool) (*tokenData, error) {
	log.Println("[INFO] In the accessToken")

	// Service Account client-credentials takes priority over the refresh
	// token flow, mirroring the esc CLI.
	if c.clientSecret != "" {
		return c.serviceAccountToken(force)
	}

	if c.tokenStore.exists(c.audience) && !force {
		tokenData, err := c.tokenStore.get(c.audience)
		if err != nil {
			return nil, fmt.Errorf("error getting token from store: %w", err)
		}

		if _, ok := tokenData.AccessToken.IsValid(); ok {
			return tokenData, nil
		}
	}

	log.Println("[INFO] authenticating via refresh token")

	// Do the refresh
	idpURL := *c.idpURL
	idpURL.Path = "/oauth/token"

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", c.clientID)
	form.Set("refresh_token", c.refreshToken)

	resp, err := c.httpClient.PostForm(idpURL.String(), form)
	if err != nil {
		return nil, fmt.Errorf("error requesting access token: %w", err)
	}
	defer closeIgnoreError(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error %d requesting access token", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	result := tokenData{}
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing IDP response: %w", err)
	}

	err = c.tokenStore.put(c.audience, result)
	if err != nil {
		return nil, fmt.Errorf("error writing token to store: %s", err.Error())
	}

	return &result, err
}

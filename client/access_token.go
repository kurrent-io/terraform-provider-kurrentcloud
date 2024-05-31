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

	err = jwt.Verify(token, jwt.WithAcceptableSkew(30*time.Second))

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

func closeIgnoreError(closer io.Closer) func() {
	return func() {
		_ = closer.Close()
	}
}

func (c *Client) accessToken(force bool) (*tokenData, error) {
	log.Println("[INFO] In the accessToken")

	if c.tokenStore.exists(c.audience) && !force {
		tokenData, err := c.tokenStore.get(c.audience)
		if err != nil {
			return nil, fmt.Errorf("error getting token from store: %w", err)
		}

		if _, ok := tokenData.AccessToken.IsValid(); ok {
			return tokenData, nil
		}
	}

	log.Println("[INFO] In the IDP")

	// Do the refresh
	idpURL := *c.idpURL
	idpURL.Path = "/oauth/token"

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", c.clientID)
	form.Set("refresh_token", c.refreshToken)

	resp, err := c.retryClient.PostForm(idpURL.String(), form)
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

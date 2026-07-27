package client

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// fakeJWT builds an unsigned JWT with the given expiry. The signature part is
// filler: token validation in this package deliberately skips signature
// verification.
func fakeJWT(t *testing.T, exp time.Time) string {
	t.Helper()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))

	payloadJSON, err := json.Marshal(map[string]int64{
		"exp": exp.Unix(),
		"iat": exp.Add(-time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("marshalling payload: %v", err)
	}

	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := base64.RawURLEncoding.EncodeToString([]byte("unsigned"))

	return header + "." + payload + "." + signature
}

func TestServiceAccountTokenUsesClientCredentialsGrant(t *testing.T) {
	validToken := fakeJWT(t, time.Now().Add(time.Hour))

	hits := 0
	lastForm := map[string]string{}

	idpKit := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth2/token" {
			t.Errorf("unexpected token endpoint path %q, want /oauth2/token", r.URL.Path)
			http.NotFound(w, r)

			return
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
		}

		hits++

		for key := range r.PostForm {
			lastForm[key] = r.PostForm.Get(key)
		}

		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(tokenData{
			AccessToken: accessToken(validToken),
			ExpiresIn:   86400,
			TokenType:   "Bearer",
		})
		if err != nil {
			t.Errorf("encoding token response: %v", err)
		}
	}))
	defer idpKit.Close()

	tokenStore := t.TempDir()

	c, err := New(&Config{
		URL:            "https://api.example.test",
		IdentityKitURL: idpKit.URL,
		ClientID:       "sa-client-id",
		ClientSecret:   "sa-client-secret",
		TokenStore:     tokenStore,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	token, err := c.accessToken(false)
	if err != nil {
		t.Fatalf("accessToken: %v", err)
	}

	if string(token.AccessToken) != validToken {
		t.Errorf("unexpected access token %q", token.AccessToken)
	}

	want := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     "sa-client-id",
		"client_secret": "sa-client-secret",
	}
	if len(lastForm) != len(want) {
		t.Errorf("unexpected form fields %v, want exactly %v", lastForm, want)
	}
	for key, value := range want {
		if lastForm[key] != value {
			t.Errorf("form field %q = %q, want %q", key, lastForm[key], value)
		}
	}

	// Second call must be served from the in-memory cache.
	if _, err := c.accessToken(false); err != nil {
		t.Fatalf("accessToken (cached): %v", err)
	}
	if hits != 1 {
		t.Errorf("token endpoint hit %d times, want 1 (cached)", hits)
	}

	// Force must bypass the cache.
	if err := c.TokenRefresh(true); err != nil {
		t.Fatalf("TokenRefresh(force): %v", err)
	}
	if hits != 2 {
		t.Errorf("token endpoint hit %d times, want 2 (forced)", hits)
	}

	// Service Account tokens must never be written to the on-disk store.
	entries, err := os.ReadDir(tokenStore)
	if err != nil {
		t.Fatalf("reading token store dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("token store contains %d entries, want 0 (SA tokens are memory-only)", len(entries))
	}
}

func TestServiceAccountTokenRefetchesWhenExpired(t *testing.T) {
	expiredToken := fakeJWT(t, time.Now().Add(-time.Hour))

	hits := 0

	idpKit := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++

		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(tokenData{
			AccessToken: accessToken(expiredToken),
			ExpiresIn:   0,
			TokenType:   "Bearer",
		})
		if err != nil {
			t.Errorf("encoding token response: %v", err)
		}
	}))
	defer idpKit.Close()

	c, err := New(&Config{
		URL:            "https://api.example.test",
		IdentityKitURL: idpKit.URL,
		ClientID:       "sa-client-id",
		ClientSecret:   "sa-client-secret",
		TokenStore:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := c.accessToken(false); err != nil {
		t.Fatalf("accessToken: %v", err)
	}
	if _, err := c.accessToken(false); err != nil {
		t.Fatalf("accessToken (expired cache): %v", err)
	}

	if hits != 2 {
		t.Errorf("token endpoint hit %d times, want 2 (expired token must not be reused)", hits)
	}
}

func TestNewRequiresClientIDWithClientSecret(t *testing.T) {
	_, err := New(&Config{
		URL:          "https://api.example.test",
		ClientSecret: "sa-client-secret",
		TokenStore:   t.TempDir(),
	})
	if err == nil {
		t.Fatal("New accepted client_secret without client_id, want error")
	}
}

func TestAccessTokenPrefersServiceAccountOverRefreshToken(t *testing.T) {
	validToken := fakeJWT(t, time.Now().Add(time.Hour))

	idpKit := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(tokenData{
			AccessToken: accessToken(validToken),
			ExpiresIn:   86400,
			TokenType:   "Bearer",
		})
		if err != nil {
			t.Errorf("encoding token response: %v", err)
		}
	}))
	defer idpKit.Close()

	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("identity provider must not be called when service account credentials are set")
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	defer idp.Close()

	c, err := New(&Config{
		URL:                 "https://api.example.test",
		IdentityProviderURL: idp.URL,
		IdentityKitURL:      idpKit.URL,
		ClientID:            "sa-client-id",
		ClientSecret:        "sa-client-secret",
		RefreshToken:        "some-refresh-token",
		TokenStore:          t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := c.accessToken(false); err != nil {
		t.Fatalf("accessToken: %v", err)
	}
}

func TestAccessTokenLegacyRefreshFlowWithoutClientSecret(t *testing.T) {
	validToken := fakeJWT(t, time.Now().Add(time.Hour))

	lastForm := map[string]string{}

	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Errorf("unexpected token endpoint path %q, want /oauth/token", r.URL.Path)
			http.NotFound(w, r)

			return
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
		}

		for key := range r.PostForm {
			lastForm[key] = r.PostForm.Get(key)
		}

		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(tokenData{
			AccessToken: accessToken(validToken),
			ExpiresIn:   86400,
			TokenType:   "Bearer",
		})
		if err != nil {
			t.Errorf("encoding token response: %v", err)
		}
	}))
	defer idp.Close()

	c, err := New(&Config{
		URL:                 "https://api.example.test",
		IdentityProviderURL: idp.URL,
		ClientID:            "legacy-client-id",
		RefreshToken:        "legacy-refresh-token",
		TokenStore:          t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := c.accessToken(false); err != nil {
		t.Fatalf("accessToken: %v", err)
	}

	if lastForm["grant_type"] != "refresh_token" {
		t.Errorf("grant_type = %q, want refresh_token", lastForm["grant_type"])
	}
	if lastForm["client_id"] != "legacy-client-id" {
		t.Errorf("client_id = %q, want legacy-client-id", lastForm["client_id"])
	}
	if lastForm["refresh_token"] != "legacy-refresh-token" {
		t.Errorf("refresh_token = %q, want legacy-refresh-token", lastForm["refresh_token"])
	}
}

// newLegacyIDP returns an httptest server implementing the legacy
// /oauth/token refresh endpoint, counting hits and capturing the last form.
func newLegacyIDP(t *testing.T, token string, hits *int, lastForm map[string]string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Errorf("unexpected token endpoint path %q, want /oauth/token", r.URL.Path)
			http.NotFound(w, r)

			return
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
		}

		*hits++

		for key := range r.PostForm {
			lastForm[key] = r.PostForm.Get(key)
		}

		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(tokenData{
			AccessToken: accessToken(token),
			ExpiresIn:   86400,
			TokenType:   "Bearer",
		})
		if err != nil {
			t.Errorf("encoding token response: %v", err)
		}
	}))
}

func TestNewAllowsClientIDWithoutClientSecret(t *testing.T) {
	c, err := New(&Config{
		URL:        "https://api.example.test",
		ClientID:   "custom-legacy-client-id",
		TokenStore: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New rejected client_id without client_secret, breaking legacy configs: %v", err)
	}

	if c.clientSecret != "" {
		t.Errorf("clientSecret = %q, want empty", c.clientSecret)
	}
}

func TestAccessTokenLegacyDefaultClientID(t *testing.T) {
	validToken := fakeJWT(t, time.Now().Add(time.Hour))

	hits := 0
	lastForm := map[string]string{}

	idp := newLegacyIDP(t, validToken, &hits, lastForm)
	defer idp.Close()

	c, err := New(&Config{
		URL:                 "https://api.example.test",
		IdentityProviderURL: idp.URL,
		RefreshToken:        "legacy-refresh-token",
		TokenStore:          t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := c.accessToken(false); err != nil {
		t.Fatalf("accessToken: %v", err)
	}

	if lastForm["client_id"] != "OraYp3cFES9O8aWuQtnqi1A7m534iTwt" {
		t.Errorf("client_id = %q, want default interactive client id", lastForm["client_id"])
	}
}

func TestAccessTokenLegacyRefreshWritesTokenStore(t *testing.T) {
	validToken := fakeJWT(t, time.Now().Add(time.Hour))

	hits := 0
	lastForm := map[string]string{}

	idp := newLegacyIDP(t, validToken, &hits, lastForm)
	defer idp.Close()

	tokenStore := t.TempDir()

	c, err := New(&Config{
		URL:                 "https://api.example.test",
		IdentityProviderURL: idp.URL,
		RefreshToken:        "legacy-refresh-token",
		TokenStore:          tokenStore,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := c.accessToken(false); err != nil {
		t.Fatalf("accessToken: %v", err)
	}

	stored, err := c.tokenStore.get(c.audience)
	if err != nil {
		t.Fatalf("refreshed token was not written to the token store: %v", err)
	}
	if string(stored.AccessToken) != validToken {
		t.Errorf("stored access token %q, want %q", stored.AccessToken, validToken)
	}
}

func TestAccessTokenLegacyInvalidCachedTokenTriggersRefresh(t *testing.T) {
	validToken := fakeJWT(t, time.Now().Add(time.Hour))
	expiredToken := fakeJWT(t, time.Now().Add(-time.Hour))

	hits := 0
	lastForm := map[string]string{}

	idp := newLegacyIDP(t, validToken, &hits, lastForm)
	defer idp.Close()

	tokenStore := t.TempDir()

	c, err := New(&Config{
		URL:                 "https://api.example.test",
		IdentityProviderURL: idp.URL,
		RefreshToken:        "legacy-refresh-token",
		TokenStore:          tokenStore,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Pre-place an expired token in the store; it must not be reused.
	err = c.tokenStore.put(c.audience, tokenData{
		AccessToken: accessToken(expiredToken),
		TokenType:   "Bearer",
	})
	if err != nil {
		t.Fatalf("seeding token store: %v", err)
	}

	token, err := c.accessToken(false)
	if err != nil {
		t.Fatalf("accessToken: %v", err)
	}

	if hits != 1 {
		t.Errorf("token endpoint hit %d times, want 1 (expired cached token must trigger refresh)", hits)
	}
	if string(token.AccessToken) != validToken {
		t.Errorf("access token %q, want freshly refreshed token", token.AccessToken)
	}
}

func TestAccessTokenLegacyForceSkipsCache(t *testing.T) {
	validToken := fakeJWT(t, time.Now().Add(time.Hour))

	hits := 0
	lastForm := map[string]string{}

	idp := newLegacyIDP(t, validToken, &hits, lastForm)
	defer idp.Close()

	c, err := New(&Config{
		URL:                 "https://api.example.test",
		IdentityProviderURL: idp.URL,
		RefreshToken:        "legacy-refresh-token",
		TokenStore:          t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := c.TokenRefresh(true); err != nil {
		t.Fatalf("TokenRefresh(force): %v", err)
	}

	if hits != 1 {
		t.Errorf("token endpoint hit %d times, want 1", hits)
	}
	if lastForm["grant_type"] != "refresh_token" {
		t.Errorf("grant_type = %q, want refresh_token", lastForm["grant_type"])
	}
}

func TestAccessTokenIsUsable(t *testing.T) {
	cases := []struct {
		name  string
		token string
		want  bool
	}{
		{name: "valid", token: fakeJWT(t, time.Now().Add(time.Hour)), want: true},
		{name: "expired", token: fakeJWT(t, time.Now().Add(-time.Hour)), want: false},
		{name: "garbage", token: "not-a-jwt", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := accessToken(tc.token).isUsable(); got != tc.want {
				t.Errorf("isUsable() = %v, want %v", got, tc.want)
			}
		})
	}
}

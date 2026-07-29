package client

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwt"
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

// unwritableStorePath returns a token-store path that os.MkdirAll cannot
// create, because a regular file sits where a parent directory would need to
// be (mkdir under a file yields ENOTDIR).
func unwritableStorePath(t *testing.T) string {
	t.Helper()

	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("creating blocker file: %v", err)
	}

	return filepath.Join(blocker, "tokens")
}

// TestNewServiceAccountSkipsTokenStore proves SA mode is memory-only: New
// succeeds even when the token store cannot be created, and no directory is
// written. An empty token store path is also accepted.
func TestNewServiceAccountSkipsTokenStore(t *testing.T) {
	storePath := unwritableStorePath(t)

	c, err := New(&Config{
		URL:          "https://api.example.test",
		ClientID:     "sa-client-id",
		ClientSecret: "sa-client-secret",
		TokenStore:   storePath,
	})
	if err != nil {
		t.Fatalf("New rejected SA config over token store access: %v", err)
	}
	if _, statErr := os.Stat(storePath); statErr == nil {
		t.Error("SA mode created the token store on disk; it must be memory-only")
	}

	// Empty token store must also be accepted in SA mode.
	if _, err := New(&Config{
		URL:          "https://api.example.test",
		ClientID:     "sa-client-id",
		ClientSecret: "sa-client-secret",
	}); err != nil {
		t.Fatalf("New rejected SA config with empty token store: %v", err)
	}

	_ = c
}

// TestNewLegacyStillValidatesTokenStore guards the refresh-token path: with no
// client secret, an uncreatable token store is still a hard error.
func TestNewLegacyStillValidatesTokenStore(t *testing.T) {
	_, err := New(&Config{
		URL:          "https://api.example.test",
		RefreshToken: "legacy-refresh-token",
		TokenStore:   unwritableStorePath(t),
	})
	if err == nil {
		t.Fatal("New accepted an uncreatable token store in refresh-token mode, want error")
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

// signedJWT mints a JWT signed with a freshly generated RSA key (RS256) and
// returns both the compact token and the private key used to sign it.
func signedJWT(t *testing.T, exp time.Time) (string, *rsa.PrivateKey) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}

	tok := jwt.New()
	if err := tok.Set(jwt.ExpirationKey, exp); err != nil {
		t.Fatalf("setting exp: %v", err)
	}

	signed, err := jwt.Sign(tok, jwa.RS256, key)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	return string(signed), key
}

// TestServiceAccountTokenPreservesPathPrefix ensures the client-credentials
// endpoint is built by appending to the configured identity_kit_url path, not
// by overwriting it, so a base URL behind a reverse proxy path prefix works.
func TestServiceAccountTokenPreservesPathPrefix(t *testing.T) {
	validToken := fakeJWT(t, time.Now().Add(time.Hour))

	var gotPath string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path

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
	defer backend.Close()

	c, err := New(&Config{
		URL:            "https://api.example.test",
		IdentityKitURL: backend.URL + "/auth/kit",
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

	if gotPath != "/auth/kit/oauth2/token" {
		t.Errorf("token endpoint path = %q, want /auth/kit/oauth2/token (prefix must be preserved)", gotPath)
	}
}

// TestIsUsableIgnoresSignature is the living proof that isUsable performs no
// signature verification. It signs a token with a real RSA key that the
// provider has never seen, then asserts:
//
//  1. isUsable accepts the token despite the unknown signing key; and
//  2. a genuine verifier rejects that same key — so the signature is real and
//     enforceable, and isUsable's acceptance is a deliberate skip, not an
//     accident of a degenerate signature.
func TestIsUsableIgnoresSignature(t *testing.T) {
	signed, signingKey := signedJWT(t, time.Now().Add(time.Hour))

	if !accessToken(signed).isUsable() {
		t.Fatal("isUsable rejected a non-expired token; it must accept regardless of signer")
	}

	// A real verifier keyed to a DIFFERENT public key must reject the token,
	// proving the signature carries meaning that isUsable deliberately skips.
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating second RSA key: %v", err)
	}
	if _, err := jwt.ParseString(signed, jwt.WithVerify(jwa.RS256, &otherKey.PublicKey)); err == nil {
		t.Fatal("expected signature verification against the wrong key to fail")
	}

	// Sanity: the same verifier keyed to the correct public key accepts it.
	if _, err := jwt.ParseString(signed, jwt.WithVerify(jwa.RS256, &signingKey.PublicKey)); err != nil {
		t.Fatalf("signature verification against the correct key failed: %v", err)
	}
}

// TestIsUsableRejectsSignedButExpired confirms isUsable still enforces expiry
// even for a validly signed token — signature is skipped, claims are not.
func TestIsUsableRejectsSignedButExpired(t *testing.T) {
	signed, _ := signedJWT(t, time.Now().Add(-time.Hour))

	if accessToken(signed).isUsable() {
		t.Fatal("isUsable accepted an expired token")
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

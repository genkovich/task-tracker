//go:build integration

package ports_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/genkovich/task-tracker/api/internal/modules/auth"
	"github.com/genkovich/task-tracker/api/internal/modules/auth/app"
	"github.com/genkovich/task-tracker/api/internal/modules/auth/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/auth/ports"
	"github.com/genkovich/task-tracker/api/internal/modules/user"
	"github.com/genkovich/task-tracker/api/internal/platform/authmw"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/internal/platform/storage"
	"github.com/genkovich/task-tracker/api/internal/server"
	"github.com/genkovich/task-tracker/api/migrations"
)

const testJWTSecret = "test-secret-key-for-integration-tests"

// mockGoogleServer creates an httptest.Server that mimics Google OAuth endpoints.
func mockGoogleServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// Token endpoint.
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		code := r.FormValue("code")
		if code == "" || code == "invalid_code" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error": "invalid_grant"}`)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token": "mock-google-access-token", "token_type": "Bearer"}`)
	})

	// UserInfo endpoint.
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer mock-google-access-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"sub": "google-user-123",
			"email": "testuser@gmail.com",
			"name": "Test User",
			"given_name": "Test",
			"family_name": "User",
			"picture": "https://example.com/photo.jpg",
			"email_verified": true
		}`)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func setupAuthServer(t *testing.T) (*httptest.Server, *httptest.Server) {
	t.Helper()

	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	googleSrv := mockGoogleServer(t)

	userRepo := infra.NewUserRepository(c.DB)
	tokenService := infra.NewJWTService(testJWTSecret)
	googleProvider := infra.NewGoogleOAuthProvider(infra.GoogleOAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/api/v1/auth/google/callback",
		TokenURL:     googleSrv.URL + "/token",
		UserInfoURL:  googleSrv.URL + "/userinfo",
	})

	svc := app.NewService(userRepo, tokenService, googleProvider, "http://localhost:5173")
	codeStore := infra.NewAuthCodeStore()
	t.Cleanup(codeStore.Stop)
	authHandler := ports.NewHandler(svc, false, codeStore)
	userHandler := user.New(c.DB, storage.NewLocal(t.TempDir(), "/files/avatars"))

	tokenValidator := auth.NewTokenValidator(testJWTSecret)
	authMW := authmw.Middleware(tokenValidator)

	s := server.New(c.DB, "http://localhost:5173", authMW, userHandler, authHandler)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	return ts, googleSrv
}

// startOAuthFlow calls GET /auth/google (without following redirect)
// and returns the state cookie and the state value extracted from the redirect Location.
func startOAuthFlow(t *testing.T, tsURL string) (*http.Cookie, string) {
	t.Helper()

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(tsURL + "/api/v1/auth/google")
	if err != nil {
		t.Fatalf("start OAuth flow: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	locURL, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	state := locURL.Query().Get("state")
	if state == "" {
		t.Fatal("state not found in redirect Location URL")
	}

	var stateCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "oauth_state" {
			stateCookie = c
		}
	}
	if stateCookie == nil {
		t.Fatal("oauth_state cookie not found in GET /auth/google response")
	}

	return stateCookie, state
}

// doCallback performs the OAuth callback with proper state cookie.
// Returns the HTTP response (redirect not followed).
func doCallback(t *testing.T, tsURL, code string) *http.Response {
	t.Helper()

	stateCookie, state := startOAuthFlow(t, tsURL)

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	callbackURL := fmt.Sprintf("%s/api/v1/auth/google/callback?code=%s&state=%s", tsURL, code, url.QueryEscape(state))
	req, err := http.NewRequest(http.MethodGet, callbackURL, nil)
	if err != nil {
		t.Fatalf("build callback request: %v", err)
	}
	req.AddCookie(stateCookie)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("callback request: %v", err)
	}
	return resp
}

type tokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type currentUserResp struct {
	ID         string  `json:"id"`
	Email      string  `json:"email"`
	FirstName  *string `json:"first_name"`
	LastName   *string `json:"last_name"`
	AvatarURL  *string `json:"avatar_url"`
	Role       string  `json:"role"`
	Position   *string `json:"position"`
	Department *string `json:"department"`
	Bio        *string `json:"bio"`
	Timezone   *string `json:"timezone"`
}

type errorResp struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestGoogleAuthURL(t *testing.T) {
	ts, _ := setupAuthServer(t)

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(ts.URL + "/api/v1/auth/google")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusFound)
	}

	// Check that oauth_state cookie is set.
	var stateCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "oauth_state" {
			stateCookie = c
		}
	}
	if stateCookie == nil {
		t.Fatal("expected oauth_state cookie")
	}
	if stateCookie.Value == "" {
		t.Fatal("expected non-empty oauth_state cookie value")
	}
	if !stateCookie.HttpOnly {
		t.Fatal("expected HttpOnly cookie")
	}

	location := resp.Header.Get("Location")
	if location == "" {
		t.Fatal("expected non-empty Location header")
	}
	if !strings.Contains(location, "client_id=test-client-id") {
		t.Fatalf("Location should contain client_id, got: %s", location)
	}
	if !strings.Contains(location, "scope=openid+email+profile") {
		t.Fatalf("Location should contain scope, got: %s", location)
	}
	if !strings.Contains(location, "state="+url.QueryEscape(stateCookie.Value)) {
		t.Fatalf("Location state should match cookie, got: %s", location)
	}
}

func TestGoogleCallbackSuccess(t *testing.T) {
	ts, _ := setupAuthServer(t)

	resp := doCallback(t, ts.URL, "valid_code")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusFound)
	}

	location := resp.Header.Get("Location")
	if !strings.HasPrefix(location, "http://localhost:5173/auth/callback?code=") {
		t.Fatalf("unexpected redirect location: %s", location)
	}
	// Should NOT contain tokens in the URL.
	if strings.Contains(location, "access_token=") {
		t.Fatalf("redirect should NOT contain access_token: %s", location)
	}
	if strings.Contains(location, "refresh_token=") {
		t.Fatalf("redirect should NOT contain refresh_token: %s", location)
	}
}

func TestGoogleCallbackInvalidState(t *testing.T) {
	ts, _ := setupAuthServer(t)

	// Call callback with wrong state (no matching cookie).
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(ts.URL + "/api/v1/auth/google/callback?code=valid_code&state=wrong-state")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var errBody errorResp
	decodeJSON(t, resp, &errBody)
	if errBody.Error.Code != "auth.invalid_state" {
		t.Fatalf("error code: got %q, want %q", errBody.Error.Code, "auth.invalid_state")
	}
}

func TestGoogleCallbackMissingCode(t *testing.T) {
	ts, _ := setupAuthServer(t)

	// Need valid state cookie to get past state validation.
	stateCookie, state := startOAuthFlow(t, ts.URL)

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/api/v1/auth/google/callback?state=%s", ts.URL, url.QueryEscape(state)), nil)
	req.AddCookie(stateCookie)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var errBody errorResp
	decodeJSON(t, resp, &errBody)
	if errBody.Error.Code != "validation.code_required" {
		t.Fatalf("error code: got %q, want %q", errBody.Error.Code, "validation.code_required")
	}
}

func TestRefreshTokenSuccess(t *testing.T) {
	ts, _ := setupAuthServer(t)

	// First, do the callback + exchange to get tokens.
	tokens := doCallbackAndExchange(t, ts.URL, "valid_code")

	// Refresh.
	body, _ := json.Marshal(map[string]string{"refresh_token": tokens.RefreshToken})
	refreshResp, err := http.Post(ts.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("refresh request: %v", err)
	}

	if refreshResp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want %d", refreshResp.StatusCode, http.StatusOK)
	}

	var tok tokenResp
	decodeJSON(t, refreshResp, &tok)

	if tok.AccessToken == "" {
		t.Fatal("expected non-empty access_token")
	}
	if tok.RefreshToken == "" {
		t.Fatal("expected non-empty refresh_token")
	}
	if tok.ExpiresIn <= 0 {
		t.Fatal("expected positive expires_in")
	}
}

func TestRefreshTokenInvalid(t *testing.T) {
	ts, _ := setupAuthServer(t)

	body, _ := json.Marshal(map[string]string{"refresh_token": "invalid-token"})
	resp, err := http.Post(ts.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	var errBody errorResp
	decodeJSON(t, resp, &errBody)
	if errBody.Error.Code != "auth.invalid_token" {
		t.Fatalf("error code: got %q, want %q", errBody.Error.Code, "auth.invalid_token")
	}
}

func TestMeWithValidToken(t *testing.T) {
	ts, _ := setupAuthServer(t)

	// Get tokens via callback + exchange.
	tokens := doCallbackAndExchange(t, ts.URL, "valid_code")

	// GET /auth/me.
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	meResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("me request: %v", err)
	}

	if meResp.StatusCode != http.StatusOK {
		var errBody errorResp
		decodeJSON(t, meResp, &errBody)
		t.Fatalf("got status %d, want %d, error: %+v", meResp.StatusCode, http.StatusOK, errBody)
	}

	var me currentUserResp
	decodeJSON(t, meResp, &me)

	if me.Email != "testuser@gmail.com" {
		t.Fatalf("email: got %q, want %q", me.Email, "testuser@gmail.com")
	}
	if me.Role != "member" {
		t.Fatalf("role: got %q, want %q", me.Role, "member")
	}
	if me.FirstName == nil || *me.FirstName != "Test" {
		t.Fatalf("first_name: got %v, want %q", me.FirstName, "Test")
	}
	if me.LastName == nil || *me.LastName != "User" {
		t.Fatalf("last_name: got %v, want %q", me.LastName, "User")
	}

	// Profile fields should be null for a new user.
	if me.Position != nil {
		t.Fatalf("position: got %v, want nil", me.Position)
	}
	if me.Department != nil {
		t.Fatalf("department: got %v, want nil", me.Department)
	}
	if me.Bio != nil {
		t.Fatalf("bio: got %v, want nil", me.Bio)
	}
	if me.Timezone != nil {
		t.Fatalf("timezone: got %v, want nil", me.Timezone)
	}
}

func TestMeWithoutToken(t *testing.T) {
	ts, _ := setupAuthServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/auth/me")
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	var errBody errorResp
	decodeJSON(t, resp, &errBody)
	if errBody.Error.Code != "auth.missing_token" {
		t.Fatalf("error code: got %q, want %q", errBody.Error.Code, "auth.missing_token")
	}
}

func TestFullFlow(t *testing.T) {
	ts, _ := setupAuthServer(t)

	// 1. Callback + exchange → get tokens.
	tokens := doCallbackAndExchange(t, ts.URL, "valid_code")

	// 2. /me with access token.
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	meResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("me: got status %d, want %d", meResp.StatusCode, http.StatusOK)
	}
	var me currentUserResp
	decodeJSON(t, meResp, &me)
	if me.Email != "testuser@gmail.com" {
		t.Fatalf("email: got %q", me.Email)
	}

	// 3. Refresh → new tokens.
	body, _ := json.Marshal(map[string]string{"refresh_token": tokens.RefreshToken})
	refreshResp, err := http.Post(ts.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	var newTokens tokenResp
	decodeJSON(t, refreshResp, &newTokens)

	// 4. /me with new access token.
	req2, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/auth/me", nil)
	req2.Header.Set("Authorization", "Bearer "+newTokens.AccessToken)
	meResp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("me after refresh: %v", err)
	}
	if meResp2.StatusCode != http.StatusOK {
		t.Fatalf("me after refresh: got status %d", meResp2.StatusCode)
	}
	var me2 currentUserResp
	decodeJSON(t, meResp2, &me2)
	if me2.Email != "testuser@gmail.com" {
		t.Fatalf("email after refresh: got %q", me2.Email)
	}
}

func TestCallbackCreatesUserOnce(t *testing.T) {
	ts, _ := setupAuthServer(t)

	// First callback + exchange — creates user.
	tokens1 := doCallbackAndExchange(t, ts.URL, "valid_code")

	// Second callback + exchange — finds existing user by google_id.
	tokens2 := doCallbackAndExchange(t, ts.URL, "valid_code")

	// Both should work and return tokens for the same user.
	req1, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/auth/me", nil)
	req1.Header.Set("Authorization", "Bearer "+tokens1.AccessToken)
	me1Resp, _ := http.DefaultClient.Do(req1)
	var me1 currentUserResp
	decodeJSON(t, me1Resp, &me1)

	req2, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/auth/me", nil)
	req2.Header.Set("Authorization", "Bearer "+tokens2.AccessToken)
	me2Resp, _ := http.DefaultClient.Do(req2)
	var me2 currentUserResp
	decodeJSON(t, me2Resp, &me2)

	if me1.ID != me2.ID {
		t.Fatalf("expected same user ID, got %q and %q", me1.ID, me2.ID)
	}
}

func TestUsersWithoutToken(t *testing.T) {
	ts, _ := setupAuthServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/users")
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	var errBody errorResp
	decodeJSON(t, resp, &errBody)
	if errBody.Error.Code != "auth.missing_token" {
		t.Fatalf("error code: got %q, want %q", errBody.Error.Code, "auth.missing_token")
	}
}

func TestUsersWithValidToken(t *testing.T) {
	ts, _ := setupAuthServer(t)

	// Get tokens via callback + exchange.
	tokens := doCallbackAndExchange(t, ts.URL, "valid_code")

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	usersResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("users request: %v", err)
	}
	defer usersResp.Body.Close()

	if usersResp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want %d", usersResp.StatusCode, http.StatusOK)
	}
}

// extractCode extracts the one-time code from the redirect URL query params.
func extractCode(t *testing.T, rawURL string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	code := u.Query().Get("code")
	if code == "" {
		t.Fatalf("code not found in redirect URL: %s", rawURL)
	}
	return code
}

// exchangeCode calls POST /auth/exchange with the one-time code and returns tokens.
func exchangeCode(t *testing.T, tsURL, code string) tokenResp {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"code": code})
	resp, err := http.Post(tsURL+"/api/v1/auth/exchange", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("exchange request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		var errBody errorResp
		decodeJSON(t, resp, &errBody)
		t.Fatalf("exchange: got status %d, error: %+v", resp.StatusCode, errBody)
	}
	var tok tokenResp
	decodeJSON(t, resp, &tok)
	return tok
}

// doCallbackAndExchange performs the full OAuth callback + code exchange flow.
func doCallbackAndExchange(t *testing.T, tsURL, googleCode string) tokenResp {
	t.Helper()
	resp := doCallback(t, tsURL, googleCode)
	resp.Body.Close()
	code := extractCode(t, resp.Header.Get("Location"))
	return exchangeCode(t, tsURL, code)
}

func TestExchangeCodeInvalid(t *testing.T) {
	ts, _ := setupAuthServer(t)

	body, _ := json.Marshal(map[string]string{"code": "nonexistent-code"})
	resp, err := http.Post(ts.URL+"/api/v1/auth/exchange", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	var errBody errorResp
	decodeJSON(t, resp, &errBody)
	if errBody.Error.Code != "auth.invalid_exchange_code" {
		t.Fatalf("error code: got %q, want %q", errBody.Error.Code, "auth.invalid_exchange_code")
	}
}

func TestExchangeCodeDoubleUse(t *testing.T) {
	ts, _ := setupAuthServer(t)

	resp := doCallback(t, ts.URL, "valid_code")
	resp.Body.Close()
	code := extractCode(t, resp.Header.Get("Location"))

	// First exchange succeeds.
	tok := exchangeCode(t, ts.URL, code)
	if tok.AccessToken == "" {
		t.Fatal("expected non-empty access_token on first exchange")
	}

	// Second exchange with same code fails.
	body, _ := json.Marshal(map[string]string{"code": code})
	resp2, err := http.Post(ts.URL+"/api/v1/auth/exchange", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("second exchange request: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", resp2.StatusCode, http.StatusUnauthorized)
	}
}

func TestExchangeCodeMissingBody(t *testing.T) {
	ts, _ := setupAuthServer(t)

	body, _ := json.Marshal(map[string]string{"code": ""})
	resp, err := http.Post(ts.URL+"/api/v1/auth/exchange", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var errBody errorResp
	decodeJSON(t, resp, &errBody)
	if errBody.Error.Code != "validation.code_required" {
		t.Fatalf("error code: got %q, want %q", errBody.Error.Code, "validation.code_required")
	}
}

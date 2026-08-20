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
	"testing"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/auth"
	authinfra "github.com/genkovich/task-tracker/api/internal/modules/auth/infra"
	authports "github.com/genkovich/task-tracker/api/internal/modules/auth/ports"
	"github.com/genkovich/task-tracker/api/internal/modules/user/app"
	"github.com/genkovich/task-tracker/api/internal/modules/user/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/user/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/authmw"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/internal/platform/storage"
	"github.com/genkovich/task-tracker/api/internal/server"
	"github.com/genkovich/task-tracker/api/migrations"

	authapp "github.com/genkovich/task-tracker/api/internal/modules/auth/app"
)

const testJWTSecret = "test-secret-key-for-user-integration-tests"

// seeded admin from migration 000003.
var seedAdminID = uuid.MustParse("019a0000-0000-7000-8000-000000000001")

func setupServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := infra.NewPostgresUserRepository(c.DB)
	svc := app.NewService(repo)
	localSt := storage.NewLocal(t.TempDir(), "/files/avatars")
	handler := ports.NewHandler(svc, localSt)

	tokenValidator := auth.NewTokenValidator(testJWTSecret)
	authMW := authmw.Middleware(tokenValidator)

	s := server.New(c.DB, "http://localhost:5173", authMW, handler)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	adminToken := generateJWT(t, seedAdminID, "admin@example.com", "admin")

	return ts, adminToken
}

func generateJWT(t *testing.T, userID uuid.UUID, email, role string) string {
	t.Helper()
	jwtSvc := authinfra.NewJWTService(testJWTSecret)
	pair, err := jwtSvc.Generate(userID, email, role)
	if err != nil {
		t.Fatalf("generate JWT: %v", err)
	}
	return pair.AccessToken
}

// --- HTTP helpers ---

func postJSONAuth(ts *httptest.Server, path string, body any, token string) *http.Response {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func putJSONAuth(ts *httptest.Server, path string, body any, token string) *http.Response {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func putJSON(ts *httptest.Server, path string, body any) *http.Response {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func getAuth(ts *httptest.Server, path, token string) *http.Response {
	req, _ := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func deleteAuth(ts *httptest.Server, path, token string) *http.Response {
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// --- Response types ---

type userResp struct {
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
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type listResp struct {
	Users   []userResp `json:"users"`
	HasNext bool       `json:"has_next"`
	HasPrev bool       `json:"has_prev"`
}

type errorResp struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func strPtr(s string) *string { return &s }

// --- CRUD tests (admin token) ---

func TestCreateAndGet(t *testing.T) {
	ts, adminToken := setupServer(t)

	// Create user.
	resp := postJSONAuth(ts, "/api/v1/users", map[string]any{
		"email":      "alice@example.com",
		"first_name": "Alice",
		"last_name":  "Smith",
	}, adminToken)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: got status %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var created userResp
	decodeJSON(t, resp, &created)
	if created.Email != "alice@example.com" {
		t.Fatalf("email: got %q, want %q", created.Email, "alice@example.com")
	}
	if created.Role != "member" {
		t.Fatalf("role: got %q, want %q", created.Role, "member")
	}
	if created.ID == "" {
		t.Fatal("expected non-empty id")
	}

	// Get by ID.
	getResp := getAuth(ts, "/api/v1/users/"+created.ID, adminToken)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get: got status %d, want %d", getResp.StatusCode, http.StatusOK)
	}

	var got userResp
	decodeJSON(t, getResp, &got)
	if got.ID != created.ID {
		t.Fatalf("id: got %q, want %q", got.ID, created.ID)
	}
	if *got.FirstName != "Alice" {
		t.Fatalf("first_name: got %q, want %q", *got.FirstName, "Alice")
	}
}

func TestCreateDuplicateEmail(t *testing.T) {
	ts, adminToken := setupServer(t)

	body := map[string]any{"email": "dup@example.com"}
	resp := postJSONAuth(ts, "/api/v1/users", body, adminToken)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first create: got status %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp2 := postJSONAuth(ts, "/api/v1/users", body, adminToken)
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate: got status %d, want %d", resp2.StatusCode, http.StatusConflict)
	}

	var errBody errorResp
	decodeJSON(t, resp2, &errBody)
	if errBody.Error.Code != "user.email_already_exists" {
		t.Fatalf("error code: got %q, want %q", errBody.Error.Code, "user.email_already_exists")
	}
}

func TestCreateMissingEmail(t *testing.T) {
	ts, adminToken := setupServer(t)

	resp := postJSONAuth(ts, "/api/v1/users", map[string]any{"first_name": "No Email"}, adminToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
	resp.Body.Close()
}

func TestGetNotFound(t *testing.T) {
	ts, adminToken := setupServer(t)

	resp := getAuth(ts, "/api/v1/users/01961234-5678-7000-8000-000000000000", adminToken)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	var errBody errorResp
	decodeJSON(t, resp, &errBody)
	if errBody.Error.Code != "user.not_found" {
		t.Fatalf("error code: got %q, want %q", errBody.Error.Code, "user.not_found")
	}
}

func TestUpdate(t *testing.T) {
	ts, adminToken := setupServer(t)

	// Create.
	resp := postJSONAuth(ts, "/api/v1/users", map[string]any{
		"email": "update@example.com",
		"role":  "member",
	}, adminToken)
	var created userResp
	decodeJSON(t, resp, &created)

	// Update.
	resp2 := putJSONAuth(ts, "/api/v1/users/"+created.ID, map[string]any{
		"email":      "updated@example.com",
		"first_name": "Updated",
		"role":       "admin",
	}, adminToken)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("update: got status %d, want %d", resp2.StatusCode, http.StatusOK)
	}

	var updated userResp
	decodeJSON(t, resp2, &updated)
	if updated.Email != "updated@example.com" {
		t.Fatalf("email: got %q, want %q", updated.Email, "updated@example.com")
	}
	if updated.Role != "admin" {
		t.Fatalf("role: got %q, want %q", updated.Role, "admin")
	}

	// Verify via GET.
	getResp := getAuth(ts, "/api/v1/users/"+created.ID, adminToken)
	var got userResp
	decodeJSON(t, getResp, &got)
	if got.Email != "updated@example.com" {
		t.Fatalf("get after update: email %q", got.Email)
	}
}

func TestDelete(t *testing.T) {
	ts, adminToken := setupServer(t)

	// Create.
	resp := postJSONAuth(ts, "/api/v1/users", map[string]any{"email": "delete@example.com"}, adminToken)
	var created userResp
	decodeJSON(t, resp, &created)

	// Delete.
	delResp := deleteAuth(ts, "/api/v1/users/"+created.ID, adminToken)
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: got status %d, want %d", delResp.StatusCode, http.StatusNoContent)
	}
	delResp.Body.Close()

	// Verify gone.
	getResp := getAuth(ts, "/api/v1/users/"+created.ID, adminToken)
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete: got status %d, want %d", getResp.StatusCode, http.StatusNotFound)
	}
	getResp.Body.Close()
}

func TestDeleteNotFound(t *testing.T) {
	ts, adminToken := setupServer(t)

	resp := deleteAuth(ts, "/api/v1/users/01961234-5678-7000-8000-000000000000", adminToken)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
	resp.Body.Close()
}

func TestListWithPagination(t *testing.T) {
	ts, adminToken := setupServer(t)

	// Seed admin already exists from migration. Create 3 more.
	for _, email := range []string{"page1@test.com", "page2@test.com", "page3@test.com"} {
		resp := postJSONAuth(ts, "/api/v1/users", map[string]any{"email": email}, adminToken)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create %s: got status %d", email, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// First page: limit=2.
	resp := getAuth(ts, "/api/v1/users?limit=2", adminToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: got status %d", resp.StatusCode)
	}
	var page1 listResp
	decodeJSON(t, resp, &page1)

	if len(page1.Users) != 2 {
		t.Fatalf("page1: got %d users, want 2", len(page1.Users))
	}
	if !page1.HasNext {
		t.Fatal("page1: expected has_next=true")
	}
	if page1.HasPrev {
		t.Fatal("page1: expected has_prev=false")
	}

	// Next page using after cursor.
	lastID := page1.Users[len(page1.Users)-1].ID
	resp2 := getAuth(ts, "/api/v1/users?limit=2&after="+lastID, adminToken)
	var page2 listResp
	decodeJSON(t, resp2, &page2)

	if len(page2.Users) == 0 {
		t.Fatal("page2: expected users")
	}
	if !page2.HasPrev {
		t.Fatal("page2: expected has_prev=true")
	}

	// Prev page using before cursor.
	firstID := page2.Users[0].ID
	resp3 := getAuth(ts, "/api/v1/users?limit=2&before="+firstID, adminToken)
	var page3 listResp
	decodeJSON(t, resp3, &page3)

	if len(page3.Users) == 0 {
		t.Fatal("page3: expected users on prev page")
	}
	if !page3.HasNext {
		t.Fatal("page3: expected has_next=true on prev page")
	}
}

func TestSeedAdminExists(t *testing.T) {
	ts, adminToken := setupServer(t)

	// Migration 000003 seeds admin. Verify via list.
	resp := getAuth(ts, "/api/v1/users?limit=100", adminToken)
	var list listResp
	decodeJSON(t, resp, &list)

	found := false
	for _, u := range list.Users {
		if u.Email == "admin@example.com" && u.Role == "admin" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected seeded admin user not found")
	}
}

// --- Forbidden tests (member token) ---

func TestCreateUserAsMember_Forbidden(t *testing.T) {
	ts, _ := setupServer(t)

	memberID, _ := uuid.NewV7()
	memberToken := generateJWT(t, memberID, "member@example.com", "member")

	resp := postJSONAuth(ts, "/api/v1/users", map[string]any{
		"email": "new@example.com",
	}, memberToken)

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusForbidden)
	}

	var errBody errorResp
	decodeJSON(t, resp, &errBody)
	if errBody.Error.Code != "user.forbidden" {
		t.Fatalf("error code: got %q, want %q", errBody.Error.Code, "user.forbidden")
	}
}

func TestUpdateUserAsMember_Forbidden(t *testing.T) {
	ts, adminToken := setupServer(t)

	// Create a user as admin first.
	resp := postJSONAuth(ts, "/api/v1/users", map[string]any{
		"email": "target@example.com",
	}, adminToken)
	var created userResp
	decodeJSON(t, resp, &created)

	// Try to update as member.
	memberID, _ := uuid.NewV7()
	memberToken := generateJWT(t, memberID, "member@example.com", "member")

	resp2 := putJSONAuth(ts, "/api/v1/users/"+created.ID, map[string]any{
		"email": "hacked@example.com",
		"role":  "admin",
	}, memberToken)

	if resp2.StatusCode != http.StatusForbidden {
		t.Fatalf("got status %d, want %d", resp2.StatusCode, http.StatusForbidden)
	}

	var errBody errorResp
	decodeJSON(t, resp2, &errBody)
	if errBody.Error.Code != "user.forbidden" {
		t.Fatalf("error code: got %q, want %q", errBody.Error.Code, "user.forbidden")
	}
}

func TestDeleteUserAsMember_Forbidden(t *testing.T) {
	ts, adminToken := setupServer(t)

	// Create a user as admin first.
	resp := postJSONAuth(ts, "/api/v1/users", map[string]any{
		"email": "victim@example.com",
	}, adminToken)
	var created userResp
	decodeJSON(t, resp, &created)

	// Try to delete as member.
	memberID, _ := uuid.NewV7()
	memberToken := generateJWT(t, memberID, "member@example.com", "member")

	delResp := deleteAuth(ts, "/api/v1/users/"+created.ID, memberToken)
	if delResp.StatusCode != http.StatusForbidden {
		t.Fatalf("got status %d, want %d", delResp.StatusCode, http.StatusForbidden)
	}

	var errBody errorResp
	decodeJSON(t, delResp, &errBody)
	if errBody.Error.Code != "user.forbidden" {
		t.Fatalf("error code: got %q, want %q", errBody.Error.Code, "user.forbidden")
	}

	// Verify user still exists.
	getResp := getAuth(ts, "/api/v1/users/"+created.ID, adminToken)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("user should still exist, got status %d", getResp.StatusCode)
	}
	getResp.Body.Close()
}

// --- Profile tests (OAuth-based, use setupAuthenticatedServer) ---

func mockGoogleServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token": "mock-google-access-token", "token_type": "Bearer"}`)
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"sub": "google-user-456",
			"email": "profile-test@gmail.com",
			"name": "Profile User",
			"given_name": "Profile",
			"family_name": "User",
			"picture": "https://example.com/photo.jpg",
			"email_verified": true
		}`)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func setupAuthenticatedServer(t *testing.T) (*httptest.Server, *httptest.Server) {
	t.Helper()

	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	googleSrv := mockGoogleServer(t)

	authUserRepo := authinfra.NewUserRepository(c.DB)
	tokenService := authinfra.NewJWTService(testJWTSecret)
	googleProvider := authinfra.NewGoogleOAuthProvider(authinfra.GoogleOAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/api/v1/auth/google/callback",
		TokenURL:     googleSrv.URL + "/token",
		UserInfoURL:  googleSrv.URL + "/userinfo",
	})

	authSvc := authapp.NewService(authUserRepo, tokenService, googleProvider, "http://localhost:5173")
	authCodeStore := authinfra.NewAuthCodeStore()
	t.Cleanup(authCodeStore.Stop)
	authHandler := authports.NewHandler(authSvc, false, authCodeStore)

	repo := infra.NewPostgresUserRepository(c.DB)
	svc := app.NewService(repo)
	localSt2 := storage.NewLocal(t.TempDir(), "/files/avatars")
	userHandler := ports.NewHandler(svc, localSt2)

	tokenValidator := auth.NewTokenValidator(testJWTSecret)
	authMW := authmw.Middleware(tokenValidator)

	s := server.New(c.DB, "http://localhost:5173", authMW, userHandler, authHandler)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	return ts, googleSrv
}

func getAccessToken(t *testing.T, tsURL string) string {
	t.Helper()

	// Start OAuth flow to get state.
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(tsURL + "/api/v1/auth/google")
	if err != nil {
		t.Fatalf("start OAuth: %v", err)
	}
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	locURL, _ := url.Parse(location)
	state := locURL.Query().Get("state")

	var stateCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "oauth_state" {
			stateCookie = c
		}
	}

	callbackURL := fmt.Sprintf("%s/api/v1/auth/google/callback?code=valid_code&state=%s", tsURL, url.QueryEscape(state))
	req, _ := http.NewRequest(http.MethodGet, callbackURL, nil)
	req.AddCookie(stateCookie)

	cbResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	defer cbResp.Body.Close()

	// The callback redirects to the frontend with a one-time exchange code;
	// swap it for tokens via POST /auth/exchange.
	locURL, err = url.Parse(cbResp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse callback redirect: %v", err)
	}
	code := locURL.Query().Get("code")
	if code == "" {
		t.Fatalf("no exchange code in redirect: %s", cbResp.Header.Get("Location"))
	}

	body, _ := json.Marshal(map[string]string{"code": code})
	exResp, err := http.Post(tsURL+"/api/v1/auth/exchange", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("exchange request: %v", err)
	}
	defer exResp.Body.Close()
	if exResp.StatusCode != http.StatusOK {
		t.Fatalf("exchange: got status %d", exResp.StatusCode)
	}

	var tokens struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(exResp.Body).Decode(&tokens); err != nil {
		t.Fatalf("decode exchange response: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Fatal("access_token not found in exchange response")
	}
	return tokens.AccessToken
}

func TestUpdateProfile(t *testing.T) {
	ts, _ := setupAuthenticatedServer(t)
	token := getAccessToken(t, ts.URL)

	resp := putJSONAuth(ts, "/api/v1/users/me", map[string]any{
		"first_name": "Updated",
		"last_name":  "Name",
		"position":   "Senior Engineer",
		"department": "Platform",
		"bio":        "Building things",
		"timezone":   "Europe/Kyiv",
	}, token)

	if resp.StatusCode != http.StatusOK {
		var errBody errorResp
		decodeJSON(t, resp, &errBody)
		t.Fatalf("update profile: got status %d, error: %+v", resp.StatusCode, errBody)
	}

	var updated userResp
	decodeJSON(t, resp, &updated)

	if updated.FirstName == nil || *updated.FirstName != "Updated" {
		t.Fatalf("first_name: got %v, want %q", updated.FirstName, "Updated")
	}
	if updated.Position == nil || *updated.Position != "Senior Engineer" {
		t.Fatalf("position: got %v, want %q", updated.Position, "Senior Engineer")
	}
	if updated.Department == nil || *updated.Department != "Platform" {
		t.Fatalf("department: got %v, want %q", updated.Department, "Platform")
	}
	if updated.Bio == nil || *updated.Bio != "Building things" {
		t.Fatalf("bio: got %v, want %q", updated.Bio, "Building things")
	}
	if updated.Timezone == nil || *updated.Timezone != "Europe/Kyiv" {
		t.Fatalf("timezone: got %v, want %q", updated.Timezone, "Europe/Kyiv")
	}
}

func TestUpdateProfileWithoutToken(t *testing.T) {
	ts, _ := setupAuthenticatedServer(t)

	resp := putJSON(ts, "/api/v1/users/me", map[string]any{
		"position": "Engineer",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	resp.Body.Close()
}

func TestUpdateProfileCannotChangeRoleOrEmail(t *testing.T) {
	ts, _ := setupAuthenticatedServer(t)
	token := getAccessToken(t, ts.URL)

	// Update profile — role and email fields are not in UpdateProfileRequest,
	// so they should be ignored and the original values preserved.
	resp := putJSONAuth(ts, "/api/v1/users/me", map[string]any{
		"first_name": "Hacker",
		"position":   "CTO",
	}, token)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update profile: got status %d", resp.StatusCode)
	}

	var updated userResp
	decodeJSON(t, resp, &updated)

	// Email should remain the original (from Google mock).
	if updated.Email != "profile-test@gmail.com" {
		t.Fatalf("email: got %q, want %q", updated.Email, "profile-test@gmail.com")
	}
	// Role should remain "member" (default for new users).
	if updated.Role != "member" {
		t.Fatalf("role: got %q, want %q", updated.Role, "member")
	}
}

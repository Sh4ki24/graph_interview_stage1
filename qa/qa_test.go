//go:build qa

// Package qa contains smoke/integration tests that call the running API.
// Run after `make compose-up` with:
//
//	go test -tags qa ./qa/... -count=1 -v
//
// The base URL defaults to http://localhost:8080 and can be overridden
// with the API_BASE_URL environment variable. Authentication uses the
// admin credentials from config (admin / admin123) plus signup-created users.
package qa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func baseURL() string {
	u := os.Getenv("API_BASE_URL")
	if u == "" {
		u = "http://localhost:8080"
	}
	return strings.TrimRight(u, "/")
}

func apiURL(path string) string { return baseURL() + path }

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newRequest(t *testing.T, method, path, token, body string) *http.Request {
	t.Helper()
	var req *http.Request
	var err error
	if body == "" {
		req, err = http.NewRequest(method, apiURL(path), nil)
	} else {
		req, err = http.NewRequest(method, apiURL(path), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func doReq(t *testing.T, method, path, token, body string) *http.Response {
	t.Helper()
	req := newRequest(t, method, path, token, body)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func decode(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return m
}

func decodeList(t *testing.T, resp *http.Response) ([]any, int, int, int) {
	t.Helper()
	defer resp.Body.Close()
	var v struct {
		Data  []any `json:"data"`
		Page  int   `json:"page"`
		Limit int   `json:"limit"`
		Total int   `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode list body: %v", err)
	}
	return v.Data, v.Page, v.Limit, v.Total
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body := make([]byte, 512)
		n, _ := resp.Body.Read(body)
		resp.Body.Close()
		t.Fatalf("expected %d, got %d (body: %s)", want, resp.StatusCode, string(body[:n]))
	}
}

// login returns a JWT for the given credentials.
func login(t *testing.T, username, password string) string {
	t.Helper()
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	resp := doReq(t, "POST", "/auth/login", "", body)
	if resp.StatusCode != http.StatusOK {
		raw := make([]byte, 512)
		n, _ := resp.Body.Read(raw)
		resp.Body.Close()
		t.Fatalf("login %s: %d (%s)", username, resp.StatusCode, string(raw[:n]))
	}
	var tok struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		resp.Body.Close()
		t.Fatalf("decode token: %v", err)
	}
	resp.Body.Close()
	if tok.Token == "" {
		t.Fatal("empty token from login")
	}
	return tok.Token
}

// signup registers a unique user and returns its token.
func signup(t *testing.T, username, password string) string {
	t.Helper()
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	resp := doReq(t, "POST", "/auth/signup", "", body)
	if resp.StatusCode != http.StatusCreated {
		raw := make([]byte, 512)
		n, _ := resp.Body.Read(raw)
		resp.Body.Close()
		t.Fatalf("signup %s: %d (%s)", username, resp.StatusCode, string(raw[:n]))
	}
	var tok struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		resp.Body.Close()
		t.Fatalf("decode token: %v", err)
	}
	resp.Body.Close()
	return tok.Token
}

// ---------------------------------------------------------------------------
// wait for API
// ---------------------------------------------------------------------------

func TestWaitForAPI(t *testing.T) {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(apiURL("/healthz"))
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			t.Log("API is ready")
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("API did not become healthy within 30s")
}

// ---------------------------------------------------------------------------
// healthz + swagger
// ---------------------------------------------------------------------------

func TestHealthz(t *testing.T) {
	resp := doReq(t, "GET", "/healthz", "", "")
	assertStatus(t, resp, 200)
	m := decode(t, resp)
	if m["status"] != "ok" {
		t.Fatalf("expected status=ok, got %v", m["status"])
	}
}

func TestSwaggerServed(t *testing.T) {
	resp := doReq(t, "GET", "/swagger/index.html", "", "")
	assertStatus(t, resp, 200)
}

// ---------------------------------------------------------------------------
// auth
// ---------------------------------------------------------------------------

func TestAuthFlow(t *testing.T) {
	username := fmt.Sprintf("qa_%d", time.Now().UnixNano())
	password := "secret123"

	// signup
	resp := doReq(t, "POST", "/auth/signup", "", fmt.Sprintf(`{"username":%q,"password":%q}`, username, password))
	assertStatus(t, resp, 201)
	m := decode(t, resp)
	if m["role"] != "user" {
		t.Fatalf("expected role=user, got %v", m["role"])
	}

	// duplicate signup -> 409
	resp = doReq(t, "POST", "/auth/signup", "", fmt.Sprintf(`{"username":%q,"password":%q}`, username, password))
	assertStatus(t, resp, 409)

	// login correct -> 200
	resp = doReq(t, "POST", "/auth/login", "", fmt.Sprintf(`{"username":%q,"password":%q}`, username, password))
	assertStatus(t, resp, 200)
	m = decode(t, resp)
	if m["token"] == "" {
		t.Fatal("expected non-empty token")
	}

	// login wrong password -> 401
	resp = doReq(t, "POST", "/auth/login", "", fmt.Sprintf(`{"username":%q,"password":%q}`, username, "wrong"))
	assertStatus(t, resp, 401)
}

// ---------------------------------------------------------------------------
// CRUD lifecycle (authenticated with JWT)
// ---------------------------------------------------------------------------

func TestCreateTask(t *testing.T) {
	user := signup(t, fmt.Sprintf("qa_create_%d", time.Now().UnixNano()), "secret123")
	admin := login(t, "admin", "admin123")

	// no auth -> 401
	resp := doReq(t, "POST", "/tasks", "", `{"title":"no auth"}`)
	assertStatus(t, resp, 401)

	// empty title -> 400
	resp = doReq(t, "POST", "/tasks", user, `{"title":""}`)
	assertStatus(t, resp, 400)

	// valid create -> 201
	resp = doReq(t, "POST", "/tasks", user, `{"title":"QA test task","description":"created by qa suite"}`)
	assertStatus(t, resp, 201)
	m := decode(t, resp)
	id, ok := m["id"].(float64)
	if !ok || int64(id) < 1 {
		t.Fatalf("expected numeric id, got %T", m["id"])
	}
	if m["status"] != "todo" {
		t.Fatalf("expected status=todo, got %v", m["status"])
	}
	t.Logf("created task id=%d (user token, admin token len=%d)", int64(id), len(admin))
}

func TestFullLifecycle(t *testing.T) {
	user := signup(t, fmt.Sprintf("qa_life_%d", time.Now().UnixNano()), "secret123")
	admin := login(t, "admin", "admin123")

	resp := doReq(t, "POST", "/tasks", user, `{"title":"lifecycle test"}`)
	assertStatus(t, resp, 201)
	m := decode(t, resp)
	id := int64(m["id"].(float64))

	// todo -> done invalid
	resp = doReq(t, "PATCH", fmt.Sprintf("/tasks/%d", id), user, `{"status":"done"}`)
	assertStatus(t, resp, 400)

	// todo -> progress -> done
	resp = doReq(t, "PATCH", fmt.Sprintf("/tasks/%d", id), user, `{"status":"progress"}`)
	assertStatus(t, resp, 200)
	resp = doReq(t, "PATCH", fmt.Sprintf("/tasks/%d", id), user, `{"status":"done"}`)
	assertStatus(t, resp, 200)
	m = decode(t, resp)
	if m["status"] != "done" {
		t.Fatalf("expected done, got %v", m["status"])
	}

	// close with user -> 403
	resp = doReq(t, "DELETE", fmt.Sprintf("/tasks/%d", id), user, "")
	assertStatus(t, resp, 403)

	// close with admin -> 200
	resp = doReq(t, "DELETE", fmt.Sprintf("/tasks/%d", id), admin, "")
	assertStatus(t, resp, 200)

	// get after close -> 404
	resp = doReq(t, "GET", fmt.Sprintf("/tasks/%d", id), user, "")
	assertStatus(t, resp, 404)
}

func TestGetTask(t *testing.T) {
	user := signup(t, fmt.Sprintf("qa_get_%d", time.Now().UnixNano()), "secret123")

	resp := doReq(t, "POST", "/tasks", user, `{"title":"get test"}`)
	assertStatus(t, resp, 201)
	m := decode(t, resp)
	id := int64(m["id"].(float64))

	resp = doReq(t, "GET", fmt.Sprintf("/tasks/%d", id), user, "")
	assertStatus(t, resp, 200)
	m = decode(t, resp)
	if m["title"] != "get test" {
		t.Fatalf("expected title=get test, got %v", m["title"])
	}

	assertStatus(t, doReq(t, "GET", "/tasks/999999", user, ""), 404)
	assertStatus(t, doReq(t, "GET", "/tasks/abc", user, ""), 400)
}

func TestOwnershipIsolation(t *testing.T) {
	alice := signup(t, fmt.Sprintf("qa_alice_%d", time.Now().UnixNano()), "secret123")
	// second user to prove isolation
	mallory := signup(t, fmt.Sprintf("qa_mallory_%d", time.Now().UnixNano()), "secret123")

	resp := doReq(t, "POST", "/tasks", alice, `{"title":"alice task"}`)
	assertStatus(t, resp, 201)
	m := decode(t, resp)
	id := int64(m["id"].(float64))

	// mallory must not see alice's task
	assertStatus(t, doReq(t, "GET", fmt.Sprintf("/tasks/%d", id), mallory, ""), 404)

	// alice still sees it
	assertStatus(t, doReq(t, "GET", fmt.Sprintf("/tasks/%d", id), alice, ""), 200)

	// mallory can't patch it
	assertStatus(t, doReq(t, "PATCH", fmt.Sprintf("/tasks/%d", id), mallory, `{"status":"progress"}`), 404)

	// admin can see it
	admin := login(t, "admin", "admin123")
	assertStatus(t, doReq(t, "GET", fmt.Sprintf("/tasks/%d", id), admin, ""), 200)
}

func TestListPagination(t *testing.T) {
	user := signup(t, fmt.Sprintf("qa_page_%d", time.Now().UnixNano()), "secret123")

	for i := 0; i < 5; i++ {
		assertStatus(t, doReq(t, "POST", "/tasks", user, fmt.Sprintf(`{"title":"page test %d"}`, i)), 201)
	}

	resp := doReq(t, "GET", "/tasks?page=1&limit=2", user, "")
	assertStatus(t, resp, 200)
	data, page, limit, total := decodeList(t, resp)
	if page != 1 || limit != 2 || total != 5 || len(data) != 2 {
		t.Fatalf("page1/limit2: page=%d limit=%d total=%d len=%d", page, limit, total, len(data))
	}

	resp = doReq(t, "GET", "/tasks?page=3&limit=2", user, "")
	assertStatus(t, resp, 200)
	data, _, _, _ = decodeList(t, resp)
	if len(data) != 1 {
		t.Fatalf("page3/limit2: expected 1 item, got %d", len(data))
	}
}

func TestListStatusFilter(t *testing.T) {
	user := signup(t, fmt.Sprintf("qa_filter_%d", time.Now().UnixNano()), "secret123")

	resp := doReq(t, "POST", "/tasks", user, `{"title":"filter 1"}`)
	assertStatus(t, resp, 201)
	m := decode(t, resp)
	id1 := int64(m["id"].(float64))

	assertStatus(t, doReq(t, "POST", "/tasks", user, `{"title":"filter 2"}`), 201)
	assertStatus(t, doReq(t, "PATCH", fmt.Sprintf("/tasks/%d", id1), user, `{"status":"progress"}`), 200)

	resp = doReq(t, "GET", "/tasks?status=progress", user, "")
	assertStatus(t, resp, 200)
	data, _, _, total := decodeList(t, resp)
	if total < 1 {
		t.Fatalf("expected >=1 progress task, got %d", total)
	}
	for _, item := range data {
		obj := item.(map[string]any)
		if obj["status"] != "progress" {
			t.Fatalf("expected progress, got %v", obj["status"])
		}
	}
}

func TestInvalidStatusFilter(t *testing.T) {
	user := login(t, "admin", "admin123")
	assertStatus(t, doReq(t, "GET", "/tasks?status=invalidstatus", user, ""), 400)
}
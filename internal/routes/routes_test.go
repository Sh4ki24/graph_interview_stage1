// Tests for the HTTP routes and their request/response schemas.
package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"todoapp/internal/auth"
	"todoapp/internal/config"
	"todoapp/internal/events"
	"todoapp/internal/routes"
	"todoapp/internal/task"
	"todoapp/internal/user"
)

const testJWTSecret = "test-secret"

func seedMockUsers(t *testing.T) *user.MockRepository {
	t.Helper()
	repo := user.NewMockRepository()
	admin := &user.User{Username: "admin", PasswordHash: "x", Role: user.RoleAdmin}
	require.NoError(t, repo.Create(context.Background(), admin))
	alice := &user.User{Username: "alice", PasswordHash: "x", Role: user.RoleUser}
	require.NoError(t, repo.Create(context.Background(), alice))
	return repo
}

// seedRouter builds a router backed by mock repositories with the admin
// (id 1) and regular user (id 2) seeded, and n tasks owned by user 2.
func seedRouter(t *testing.T, n int) http.Handler {
	t.Helper()
	auth.JWTSecret = testJWTSecret
	userRepo := seedMockUsers(t)
	repo := task.NewMockRepository()
	svc := task.NewService(repo, make(chan events.Event, 16), task.Options{})
	s := routes.New(svc, userRepo, config.Pagination{DefaultLimit: 20, MaxLimit: 100})
	for i := 0; i < n; i++ {
		_, err := svc.Create(context.Background(), 2, "task", "")
		require.NoError(t, err)
	}
	return s.Router()
}

// do sends a request. The role argument selects which bearer token to attach:
// "" (no auth), "user", or "admin".
func do(t *testing.T, h http.Handler, method, target, role, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	switch role {
	case "user":
		tok, _ := auth.GenerateToken(testJWTSecret, 2, "alice", auth.RoleUser)
		req.Header.Set("Authorization", "Bearer "+tok)
	case "admin":
		tok, _ := auth.GenerateToken(testJWTSecret, 1, "admin", auth.RoleAdmin)
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

// --- schema tests -----------------------------------------------------------

func TestCreateTaskInputSchema(t *testing.T) {
	var in routes.CreateTaskInput
	require.NoError(t, json.Unmarshal([]byte(`{"title":"Buy milk","description":"2 liters"}`), &in))
	assert.Equal(t, "Buy milk", in.Title)

	err := json.Unmarshal([]byte(`{"bad json"`), &in)
	assert.Error(t, err)
}

func TestUpdateTaskInputSchema(t *testing.T) {
	var in routes.UpdateTaskInput
	require.NoError(t, json.Unmarshal([]byte(`{"status":"progress","title":"x"}`), &in))
	require.NotNil(t, in.Status)
	assert.Equal(t, "progress", *in.Status)
	assert.Nil(t, in.Description)

	var bad routes.UpdateTaskInput
	assert.Error(t, json.Unmarshal([]byte(`{"bad json"`), &bad))
}

// --- route tests ------------------------------------------------------------

func TestCreateAndLifecycle(t *testing.T) {
	h := seedRouter(t, 0)

	// no auth header -> 401
	assert.Equal(t, http.StatusUnauthorized, do(t, h, "POST", "/tasks", "", `{"title":"x"}`).Code)

	// validation -> 400
	assert.Equal(t, http.StatusBadRequest, do(t, h, "POST", "/tasks", "user", `{"title":""}`).Code)

	// create -> 201
	w := do(t, h, "POST", "/tasks", "user", `{"title":"Buy milk","description":"2 liters"}`)
	require.Equal(t, http.StatusCreated, w.Code)

	var created map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "todo", created["status"])
	assert.Equal(t, float64(2), created["user_id"]) // owned by user 2

	// todo -> done directly: rejected
	assert.Equal(t, http.StatusBadRequest, do(t, h, "PATCH", "/tasks/1", "user", `{"status":"done"}`).Code)

	// todo -> progress -> done
	require.Equal(t, http.StatusOK, do(t, h, "PATCH", "/tasks/1", "user", `{"status":"progress"}`).Code)
	require.Equal(t, http.StatusOK, do(t, h, "PATCH", "/tasks/1", "user", `{"status":"done"}`).Code)

	// get
	w = do(t, h, "GET", "/tasks/1", "user", "")
	require.Equal(t, http.StatusOK, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "done", got["status"])

	// close (soft delete) requires admin
	assert.Equal(t, http.StatusForbidden, do(t, h, "DELETE", "/tasks/1", "user", "").Code)
	require.Equal(t, http.StatusOK, do(t, h, "DELETE", "/tasks/1", "admin", "").Code)

	// after close: not found
	assert.Equal(t, http.StatusNotFound, do(t, h, "GET", "/tasks/1", "user", "").Code)
}

func TestListPaginationAndStatusFilter(t *testing.T) {
	h := seedRouter(t, 5)

	require.Equal(t, http.StatusOK, do(t, h, "PATCH", "/tasks/1", "user", `{"status":"progress"}`).Code)
	require.Equal(t, http.StatusOK, do(t, h, "PATCH", "/tasks/2", "user", `{"status":"progress"}`).Code)

	// page 1, limit 2
	w := do(t, h, "GET", "/tasks?page=1&limit=2", "user", "")
	require.Equal(t, http.StatusOK, w.Code)
	var resp routes.ListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 5, resp.Total)
	assert.Len(t, resp.Data, 2)

	// page 3, limit 2 -> 1 item
	w = do(t, h, "GET", "/tasks?page=3&limit=2", "user", "")
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Data, 1)

	// status filter
	w = do(t, h, "GET", "/tasks?status=progress", "user", "")
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Total)
	for _, t1 := range resp.Data {
		assert.Equal(t, task.StatusProgress, t1.Status)
	}
}

func TestGetNotFound(t *testing.T) {
	h := seedRouter(t, 0)
	assert.Equal(t, http.StatusBadRequest, do(t, h, "GET", "/tasks/abc", "user", "").Code)
	assert.Equal(t, http.StatusNotFound, do(t, h, "GET", "/tasks/999", "user", "").Code)
}

func TestOwnershipIsolation(t *testing.T) {
	h := seedRouter(t, 1) // task 1 owned by user 2

	// a different user must not see or modify task 1
	other := httptest.NewRequest("GET", "/tasks/1", nil)
	otherTok, _ := auth.GenerateToken(testJWTSecret, 99, "mallory", auth.RoleUser)
	other.Header.Set("Authorization", "Bearer "+otherTok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, other)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// admin can see it
	assert.Equal(t, http.StatusOK, do(t, h, "GET", "/tasks/1", "admin", "").Code)
}

func TestSwaggerServed(t *testing.T) {
	h := seedRouter(t, 0)
	assert.Equal(t, http.StatusOK, do(t, h, "GET", "/swagger/index.html", "", "").Code)
}

func TestHealthz(t *testing.T) {
	h := seedRouter(t, 0)
	assert.Equal(t, http.StatusOK, do(t, h, "GET", "/healthz", "", "").Code)
}

func TestSignupLogin(t *testing.T) {
	h := seedRouter(t, 0)

	// signup a fresh user
	var signupBody = `{"username":"bob","password":"secret"}`
	req := httptest.NewRequest("POST", "/auth/signup", bytes.NewBufferString(signupBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp routes.TokenResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "bob", resp.Username)
	assert.Equal(t, "user", resp.Role)

	// duplicate signup -> 409
	req2 := httptest.NewRequest("POST", "/auth/signup", bytes.NewBufferString(signupBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusConflict, w2.Code)

	// login with correct credentials -> 200
	loginBody := `{"username":"bob","password":"secret"}`
	req3 := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(loginBody))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)
	var loginResp routes.TokenResponse
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &loginResp))
	assert.NotEmpty(t, loginResp.Token)

	// login with wrong password -> 401
	badLogin := `{"username":"bob","password":"wrong"}`
	req4 := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(badLogin))
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()
	h.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusUnauthorized, w4.Code)
}
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lomokwa/mc-manager/middleware"
	"github.com/lomokwa/mc-manager/services"
	"github.com/lomokwa/mc-manager/types"
)

func TestListServersHandler_EmptyRegistry(t *testing.T) {
	setupTestDB(t)

	r := newTestRouter()
	r.GET("/servers", ListServersHandler)

	req := httptest.NewRequest(http.MethodGet, "/servers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool             `json:"success"`
		Data    []serverListItem `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected an empty list before EnsureDefaultServer runs, got %+v", resp.Data)
	}
}

func TestListServersHandler_DefaultServer_NotRunning(t *testing.T) {
	setupTestDB(t)
	setupServerDir(t)
	if err := services.EnsureDefaultServer(); err != nil {
		t.Fatalf("failed to seed default server: %v", err)
	}

	r := newTestRouter()
	r.GET("/servers", ListServersHandler)

	req := httptest.NewRequest(http.MethodGet, "/servers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []serverListItem `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 1 {
		t.Fatalf("expected exactly 1 server, got %d: %+v", len(resp.Data), resp.Data)
	}

	item := resp.Data[0]
	if item.ID != services.DefaultServerID {
		t.Errorf("expected id %q, got %q", services.DefaultServerID, item.ID)
	}
	if item.Running {
		t.Error("expected running=false with no status file present")
	}
	if item.PID != 0 {
		t.Errorf("expected pid to be omitted/zero, got %d", item.PID)
	}
	if item.Since != nil {
		t.Errorf("expected since to be nil, got %v", item.Since)
	}
}

func TestListServersHandler_DefaultServer_Running(t *testing.T) {
	setupTestDB(t)
	setupServerDir(t)
	if err := services.EnsureDefaultServer(); err != nil {
		t.Fatalf("failed to seed default server: %v", err)
	}

	since := time.Now().Add(-5 * time.Minute).UTC().Truncate(time.Second)
	status := types.ServerRuntimeStatus{Running: true, PID: 4321, Since: since, Heartbeat: time.Now()}
	statusJSON, _ := json.Marshal(status)
	writeServerFile(t, ".mcmanager/status.json", string(statusJSON))

	r := newTestRouter()
	r.GET("/servers", ListServersHandler)

	req := httptest.NewRequest(http.MethodGet, "/servers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []serverListItem `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 1 {
		t.Fatalf("expected exactly 1 server, got %d", len(resp.Data))
	}

	item := resp.Data[0]
	if !item.Running {
		t.Error("expected running=true")
	}
	if item.PID != 4321 {
		t.Errorf("expected pid 4321, got %d", item.PID)
	}
	if item.Since == nil || !item.Since.Equal(since) {
		t.Errorf("expected since %v, got %v", since, item.Since)
	}
}

func TestGetServerHandler_UnknownID(t *testing.T) {
	setupTestDB(t)

	r := newTestRouter()
	r.GET("/servers/:sid", middleware.ResolveServer(), GetServerHandler)

	req := httptest.NewRequest(http.MethodGet, "/servers/does-not-exist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp types.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error == "" {
		t.Error("expected a non-empty error message")
	}
}

func TestGetServerHandler_Success(t *testing.T) {
	setupTestDB(t)
	setupServerDir(t)
	if err := services.EnsureDefaultServer(); err != nil {
		t.Fatalf("failed to seed default server: %v", err)
	}

	r := newTestRouter()
	r.GET("/servers/:sid", middleware.ResolveServer(), GetServerHandler)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+services.DefaultServerID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data serverListItem `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.ID != services.DefaultServerID {
		t.Errorf("expected id %q, got %q", services.DefaultServerID, resp.Data.ID)
	}
	if resp.Data.Name != "Default" {
		t.Errorf("expected name %q, got %q", "Default", resp.Data.Name)
	}
	if resp.Data.Running {
		t.Error("expected running=false with no status file present")
	}
}

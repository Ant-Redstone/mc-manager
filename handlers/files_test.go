package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lomokwa/mc-manager/services"
	"github.com/lomokwa/mc-manager/types"
)

func TestSafePath_TraversalContained(t *testing.T) {
	setupServerDir(t)

	// Leading "/" + filepath.Clean collapses ".." segments at the root, so a
	// traversal attempt resolves to a path safely inside the server dir
	// rather than escaping it.
	resolved, err := safePath(services.DefaultRuntime(), "../../etc/passwd")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	base, err := filepath.Abs(services.ServerDir)
	if err != nil {
		t.Fatalf("failed to resolve base dir: %v", err)
	}
	if !strings.HasPrefix(resolved, base) {
		t.Errorf("expected resolved path %q to stay within %q", resolved, base)
	}
}

func TestSafePath_ReservedControlDirDenied(t *testing.T) {
	setupServerDir(t)

	if _, err := safePath(services.DefaultRuntime(), ".mcmanager/status.json"); err == nil {
		t.Error("expected control dir path to be denied")
	}
	if _, err := safePath(services.DefaultRuntime(), ".mcmanager"); err == nil {
		t.Error("expected control dir itself to be denied")
	}
}

func TestSafePath_ValidPathResolves(t *testing.T) {
	setupServerDir(t)

	resolved, err := safePath(services.DefaultRuntime(), "world/level.dat")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.HasSuffix(resolved, "world/level.dat") {
		t.Errorf("expected resolved path to end with world/level.dat, got %q", resolved)
	}
}

// TestSafePath_TraversalDeniedUnderArbitraryRuntime proves path traversal
// stays impossible once safePath is anchored to a runtime instead of the
// fixed ServerDir constant: it's this runtime's own Dir -- not
// services.ServerDir -- that bounds every resolved path. That's what will
// keep a malicious :sid unable to reach another server's files once routes
// are namespaced in a later phase (see PLAN-multi-server.md, D4 and the
// path-traversal row of the risk table).
func TestSafePath_TraversalDeniedUnderArbitraryRuntime(t *testing.T) {
	dir := t.TempDir()
	rt := &services.ServerRuntime{ID: "other", Dir: dir}

	resolved, err := safePath(rt, "../../etc/passwd")
	if err != nil {
		t.Fatalf("expected no error (traversal is collapsed, not rejected outright), got %v", err)
	}

	base, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("failed to resolve base dir: %v", err)
	}
	if !strings.HasPrefix(resolved, base) {
		t.Errorf("expected resolved path %q to stay within this runtime's own dir %q", resolved, base)
	}

	defaultBase, err := filepath.Abs(services.ServerDir)
	if err != nil {
		t.Fatalf("failed to resolve default server dir: %v", err)
	}
	if strings.HasPrefix(resolved, defaultBase) {
		t.Errorf("resolved path %q leaked into the default server's directory %q instead of staying under %q", resolved, defaultBase, base)
	}
}

// TestSafePath_ReservedControlDirDeniedUnderArbitraryRuntime mirrors
// TestSafePath_ReservedControlDirDenied for a non-default runtime, since the
// reserved-control-dir guard re-derives its own path from rt too.
func TestSafePath_ReservedControlDirDeniedUnderArbitraryRuntime(t *testing.T) {
	rt := &services.ServerRuntime{ID: "other", Dir: t.TempDir()}

	if _, err := safePath(rt, ".mcmanager/status.json"); err == nil {
		t.Error("expected control dir path to be denied")
	}
}

func TestListFilesHandler_Success(t *testing.T) {
	setupServerDir(t)
	writeServerFile(t, "server.properties", "gamemode=survival\n")
	writeServerFile(t, "world/level.dat", "data")

	r := newTestRouter()
	r.GET("/files", ListFilesHandler)

	req := httptest.NewRequest(http.MethodGet, "/files?path=/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []fileEntry `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	names := map[string]bool{}
	for _, e := range resp.Data {
		names[e.Name] = true
	}
	if !names["server.properties"] || !names["world"] {
		t.Errorf("expected server.properties and world entries, got %+v", resp.Data)
	}
}

func TestListFilesHandler_NotFound(t *testing.T) {
	setupServerDir(t)

	r := newTestRouter()
	r.GET("/files", ListFilesHandler)

	req := httptest.NewRequest(http.MethodGet, "/files?path=/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestListFilesHandler_NotADirectory(t *testing.T) {
	setupServerDir(t)
	writeServerFile(t, "server.properties", "gamemode=survival\n")

	r := newTestRouter()
	r.GET("/files", ListFilesHandler)

	req := httptest.NewRequest(http.MethodGet, "/files?path=/server.properties", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestReadFileHandler_Success(t *testing.T) {
	setupServerDir(t)
	writeServerFile(t, "server.properties", "gamemode=survival\n")

	r := newTestRouter()
	r.GET("/files/read", ReadFileHandler)

	req := httptest.NewRequest(http.MethodGet, "/files/read?path=/server.properties", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp types.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data != "gamemode=survival\n" {
		t.Errorf("unexpected content: %+v", resp.Data)
	}
}

func TestReadFileHandler_Directory(t *testing.T) {
	setupServerDir(t)
	writeServerFile(t, "world/level.dat", "data")

	r := newTestRouter()
	r.GET("/files/read", ReadFileHandler)

	req := httptest.NewRequest(http.MethodGet, "/files/read?path=/world", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestReadFileHandler_TooLarge(t *testing.T) {
	setupServerDir(t)
	big := strings.Repeat("a", 5*1024*1024+1)
	writeServerFile(t, "big.bin", big)

	r := newTestRouter()
	r.GET("/files/read", ReadFileHandler)

	req := httptest.NewRequest(http.MethodGet, "/files/read?path=/big.bin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestWriteFileHandler_Success(t *testing.T) {
	setupServerDir(t)
	writeServerFile(t, "server.properties", "gamemode=survival\n")

	r := newTestRouter()
	r.POST("/files/write", WriteFileHandler)

	body := `{"content": "gamemode=creative\n"}`
	req := httptest.NewRequest(http.MethodPost, "/files/write?path=/server.properties", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	got, err := os.ReadFile("minecraft-server/server.properties")
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(got) != "gamemode=creative\n" {
		t.Errorf("expected written content, got %q", got)
	}
}

func TestWriteFileHandler_NotFound(t *testing.T) {
	setupServerDir(t)

	r := newTestRouter()
	r.POST("/files/write", WriteFileHandler)

	body := `{"content": "data"}`
	req := httptest.NewRequest(http.MethodPost, "/files/write?path=/missing.txt", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestDownloadFileHandler_Success(t *testing.T) {
	setupServerDir(t)
	writeServerFile(t, "server.properties", "gamemode=survival\n")

	r := newTestRouter()
	r.GET("/files/download", DownloadFileHandler)

	req := httptest.NewRequest(http.MethodGet, "/files/download?path=/server.properties", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if w.Body.String() != "gamemode=survival\n" {
		t.Errorf("unexpected content: %q", w.Body.String())
	}
}

func TestDownloadFileHandler_Directory(t *testing.T) {
	setupServerDir(t)
	writeServerFile(t, "world/level.dat", "data")

	r := newTestRouter()
	r.GET("/files/download", DownloadFileHandler)

	req := httptest.NewRequest(http.MethodGet, "/files/download?path=/world", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestUploadFileHandler_Success(t *testing.T) {
	setupServerDir(t)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "uploaded.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	fw.Write([]byte("uploaded content"))
	mw.Close()

	r := newTestRouter()
	r.POST("/files/upload", UploadFileHandler)

	req := httptest.NewRequest(http.MethodPost, "/files/upload?path=/", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	got, err := os.ReadFile("minecraft-server/uploaded.txt")
	if err != nil {
		t.Fatalf("expected uploaded file to exist: %v", err)
	}
	if string(got) != "uploaded content" {
		t.Errorf("unexpected uploaded content: %q", got)
	}
}

func TestUploadFileHandler_TargetNotDirectory(t *testing.T) {
	setupServerDir(t)
	writeServerFile(t, "server.properties", "gamemode=survival\n")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "uploaded.txt")
	fw.Write([]byte("data"))
	mw.Close()

	r := newTestRouter()
	r.POST("/files/upload", UploadFileHandler)

	req := httptest.NewRequest(http.MethodPost, "/files/upload?path=/server.properties", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteFileHandler_Success(t *testing.T) {
	setupServerDir(t)
	writeServerFile(t, "world/level.dat", "data")

	r := newTestRouter()
	r.DELETE("/files", DeleteFileHandler)

	req := httptest.NewRequest(http.MethodDelete, "/files?path=/world", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if _, err := os.Stat("minecraft-server/world"); !os.IsNotExist(err) {
		t.Error("expected world dir to be removed")
	}
}

func TestDeleteFileHandler_RefusesServerRoot(t *testing.T) {
	setupServerDir(t)

	r := newTestRouter()
	r.DELETE("/files", DeleteFileHandler)

	req := httptest.NewRequest(http.MethodDelete, "/files?path=/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteFileHandler_NotFound(t *testing.T) {
	setupServerDir(t)

	r := newTestRouter()
	r.DELETE("/files", DeleteFileHandler)

	req := httptest.NewRequest(http.MethodDelete, "/files?path=/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body=%s", w.Code, w.Body.String())
	}
}

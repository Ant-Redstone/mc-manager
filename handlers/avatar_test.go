package handlers

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lomokwa/mc-manager/services"
	"github.com/lomokwa/mc-manager/types"
)

// pngBytes builds a tiny valid PNG so tests exercise the real content-type
// sniff SaveAvatar performs, instead of a hand-rolled byte literal.
func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}
	return buf.Bytes()
}

// multipartAvatarRequest builds a POST with a single "avatar" file field.
func multipartAvatarRequest(t *testing.T, url, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("avatar", filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write form file content: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestUploadAvatarHandler_Success(t *testing.T) {
	setupTestDB(t)
	setupAvatarDir(t)
	_, r := withUser(t, "picuser1", "")
	r.POST("/me/avatar", UploadAvatarHandler)

	req := multipartAvatarRequest(t, "/me/avatar", "photo.png", pngBytes(t))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data types.User `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data.AvatarURL == "" || !strings.HasPrefix(resp.Data.AvatarURL, "/api/avatars/") {
		t.Errorf("expected avatar_url under /api/avatars/, got %q", resp.Data.AvatarURL)
	}
}

func TestUploadAvatarHandler_RejectsNonImage(t *testing.T) {
	setupTestDB(t)
	setupAvatarDir(t)
	_, r := withUser(t, "picuser2", "")
	r.POST("/me/avatar", UploadAvatarHandler)

	req := multipartAvatarRequest(t, "/me/avatar", "not-an-image.txt", []byte("just some plain text, not an image"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a non-image upload, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestUploadAvatarHandler_RejectsOversized(t *testing.T) {
	setupTestDB(t)
	setupAvatarDir(t)
	_, r := withUser(t, "picuser3", "")
	r.POST("/me/avatar", UploadAvatarHandler)

	oversized := make([]byte, services.AvatarMaxBytes+1)
	req := multipartAvatarRequest(t, "/me/avatar", "huge.png", oversized)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for an oversized upload, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestUploadAvatarHandler_ReplacesPreviousFile(t *testing.T) {
	setupTestDB(t)
	dir := setupAvatarDir(t)
	userID, r := withUser(t, "picuser4", "")
	r.POST("/me/avatar", UploadAvatarHandler)

	// First upload.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, multipartAvatarRequest(t, "/me/avatar", "first.png", pngBytes(t)))
	var first struct {
		Data types.User `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &first)

	// Second upload should replace it, both on disk and in the DB.
	w = httptest.NewRecorder()
	r.ServeHTTP(w, multipartAvatarRequest(t, "/me/avatar", "second.png", pngBytes(t)))
	var second struct {
		Data types.User `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &second)

	if second.Data.AvatarURL == first.Data.AvatarURL {
		t.Fatalf("expected a new avatar_url on replace, both were %q", first.Data.AvatarURL)
	}

	entries, err := readDirNames(t, dir)
	if err != nil {
		t.Fatalf("failed to read avatar dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected exactly one avatar file to remain for user %d, got %v", userID, entries)
	}
}

func TestDeleteAvatarHandler_RemovesFile(t *testing.T) {
	setupTestDB(t)
	dir := setupAvatarDir(t)
	_, r := withUser(t, "picuser5", "")
	r.POST("/me/avatar", UploadAvatarHandler)
	r.DELETE("/me/avatar", DeleteAvatarHandler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, multipartAvatarRequest(t, "/me/avatar", "photo.png", pngBytes(t)))
	if w.Code != http.StatusOK {
		t.Fatalf("setup: expected upload to succeed, got %d, body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/me/avatar", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	entries, err := readDirNames(t, dir)
	if err != nil {
		t.Fatalf("failed to read avatar dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected avatar file to be removed, still found %v", entries)
	}
}

func TestUpdateProfileHandler_SetsAndClearsDisplayName(t *testing.T) {
	setupTestDB(t)
	_, r := withUser(t, "nameuser1", "")
	r.PATCH("/me", UpdateProfileHandler)

	req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"display_name":"Ren"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data types.User `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.DisplayName != "Ren" {
		t.Errorf("expected display_name %q, got %q", "Ren", resp.Data.DisplayName)
	}

	// Clearing it back to "" should succeed too.
	req = httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"display_name":""}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 clearing display_name, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateProfileHandler_RejectsTooLong(t *testing.T) {
	setupTestDB(t)
	_, r := withUser(t, "nameuser2", "")
	r.PATCH("/me", UpdateProfileHandler)

	tooLong := strings.Repeat("a", 33)
	req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"display_name":"`+tooLong+`"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an over-length display name, got %d, body=%s", w.Code, w.Body.String())
	}
}

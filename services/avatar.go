package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/lomokwa/mc-manager/db"
)

// AvatarMaxBytes caps profile picture uploads. Avatars are displayed small,
// so a multi-hundred-MB "profile picture" adds nothing but disk pressure on
// the homelab box this stores them on -- 5MB comfortably fits any real photo
// or PNG/GIF avatar while keeping a careless (or malicious) upload cheap to
// reject.
const AvatarMaxBytes = 5 * 1024 * 1024 // 5MB

// avatarExtByMIME maps a sniffed content type to the extension SaveAvatar
// stores it under. Only real image formats are accepted -- the extension is
// never taken from the client-supplied filename, which also sidesteps any
// path-traversal or double-extension tricks a filename could otherwise carry.
var avatarExtByMIME = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

// SaveAvatar validates and stores an uploaded profile picture for userID,
// replacing (and deleting) any previous one, and returns the new file's name
// as recorded on the user's row. file must already be known not to exceed
// AvatarMaxBytes (the HTTP layer enforces this before the multipart body is
// even parsed) -- this only re-checks it while copying, in case a caller
// forgets to.
func SaveAvatar(userID int, file io.ReadSeeker) (string, error) {
	sniff := make([]byte, 512)
	n, err := io.ReadFull(file, sniff)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("failed to read upload: %w", err)
	}
	contentType := http.DetectContentType(sniff[:n])
	ext, ok := avatarExtByMIME[contentType]
	if !ok {
		return "", fmt.Errorf("unsupported image type %q: only JPEG, PNG, GIF, and WebP are allowed", contentType)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to read upload: %w", err)
	}

	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return "", fmt.Errorf("failed to generate filename: %w", err)
	}
	// Randomized, server-chosen filename: never derived from the client's
	// filename, and it changes on every upload so a browser can't keep
	// showing a cached, since-replaced picture at the same URL.
	filename := fmt.Sprintf("%d-%s.%s", userID, hex.EncodeToString(suffix), ext)

	if err := os.MkdirAll(AvatarDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create avatar directory: %w", err)
	}

	dest := filepath.Join(AvatarDir, filename)
	out, err := os.Create(dest)
	if err != nil {
		return "", fmt.Errorf("failed to save avatar: %w", err)
	}
	defer out.Close()

	// Belt-and-suspenders: the HTTP handler already bounds the request body,
	// but SaveAvatar shouldn't rely on that to stay true from every caller.
	limited := io.LimitReader(file, AvatarMaxBytes+1)
	written, err := io.Copy(out, limited)
	if err != nil {
		out.Close()
		os.Remove(dest)
		return "", fmt.Errorf("failed to save avatar: %w", err)
	}
	if written > AvatarMaxBytes {
		out.Close()
		os.Remove(dest)
		return "", fmt.Errorf("image too large: max %dMB", AvatarMaxBytes/(1024*1024))
	}
	if err := out.Close(); err != nil {
		os.Remove(dest)
		return "", fmt.Errorf("failed to save avatar: %w", err)
	}

	old, err := getAvatarFilename(userID)
	if err != nil {
		os.Remove(dest)
		return "", fmt.Errorf("failed to look up existing avatar: %w", err)
	}

	if _, err := db.DB.Exec("UPDATE users SET avatar_filename = ? WHERE id = ?", filename, userID); err != nil {
		os.Remove(dest)
		return "", fmt.Errorf("failed to save avatar: %w", err)
	}

	if old != "" && old != filename {
		os.Remove(filepath.Join(AvatarDir, old))
	}

	return filename, nil
}

// DeleteAvatar removes userID's stored profile picture, if any, and clears
// the column pointing at it.
func DeleteAvatar(userID int) error {
	filename, err := getAvatarFilename(userID)
	if err != nil {
		return err
	}
	if filename == "" {
		return nil
	}

	if _, err := db.DB.Exec("UPDATE users SET avatar_filename = '' WHERE id = ?", userID); err != nil {
		return err
	}

	os.Remove(filepath.Join(AvatarDir, filename))
	return nil
}

func getAvatarFilename(userID int) (string, error) {
	var filename string
	err := db.DB.QueryRow("SELECT avatar_filename FROM users WHERE id = ?", userID).Scan(&filename)
	return filename, err
}

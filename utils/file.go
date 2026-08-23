package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func DownloadFile(url, filePath string) error {
	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func WriteFile(filePath string, fileContent []byte) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(filePath, fileContent, 0644); err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}

	return nil
}

// WriteFileAtomic writes fileContent via a temp file + rename so a reader
// (or a crash mid-write) never observes a partially-written file. Prefer
// this over WriteFile for anything the running Minecraft server also reads,
// such as ops.json/whitelist.json/usercache.json.
func WriteFileAtomic(filePath string, fileContent []byte) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, fileContent, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize %s: %w", filePath, err)
	}

	return nil
}

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalPhotoFilesFiltersSupportedExtensions(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, name := range []string{"a.jpg", "b.PNG", "c.txt", filepath.Join("nested", "d.webp")} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	files, err := localPhotoFiles(dir, []string{".jpg", "png", ".webp"})
	if err != nil {
		t.Fatalf("localPhotoFiles: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d: %#v", len(files), files)
	}
}

func TestLocalPhotoFilesMissingDirIsEmpty(t *testing.T) {
	files, err := localPhotoFiles(filepath.Join(t.TempDir(), "missing"), nil)
	if err != nil {
		t.Fatalf("localPhotoFiles: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no files, got %#v", files)
	}
}

func TestLocalPhotoFilesWithFallbacksFindsPhotosFolder(t *testing.T) {
	root := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore wd: %v", err)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := os.MkdirAll("photos", 0o755); err != nil {
		t.Fatalf("mkdir photos: %v", err)
	}
	if err := os.WriteFile(filepath.Join("photos", "laya.jpg"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write photo: %v", err)
	}

	files, err := localPhotoFilesWithFallbacks(filepath.Join("data", "photos"), []string{".jpg"})
	if err != nil {
		t.Fatalf("localPhotoFilesWithFallbacks: %v", err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "laya.jpg" {
		t.Fatalf("unexpected files: %#v", files)
	}
}

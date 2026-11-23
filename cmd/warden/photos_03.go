package main

import (
	"path/filepath"
	"sort"
	"strings"
)

func localPhotoFilesWithFallbacks(configuredDir string, extensions []string) ([]string, error) {
	candidates := []string{configuredDir, "photos", filepath.Join("data", "photos")}
	seenDirs := make(map[string]struct{}, len(candidates))
	seenFiles := make(map[string]struct{})
	var out []string

	for _, dir := range candidates {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		clean := filepath.Clean(dir)
		if _, ok := seenDirs[clean]; ok {
			continue
		}
		seenDirs[clean] = struct{}{}

		files, err := localPhotoFiles(clean, extensions)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			abs, err := filepath.Abs(file)
			if err != nil {
				abs = file
			}
			if _, ok := seenFiles[abs]; ok {
				continue
			}
			seenFiles[abs] = struct{}{}
			out = append(out, file)
		}
	}
	sort.Strings(out)
	return out, nil
}

func localPhotoExtensionSet(extensions []string) map[string]struct{} {
	if len(extensions) == 0 {
		extensions = []string{".jpg", ".jpeg", ".png", ".webp", ".jfif"}
	}
	allowed := make(map[string]struct{}, len(extensions))
	for _, ext := range extensions {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		allowed[ext] = struct{}{}
	}
	return allowed
}

package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (s *PhotoStore) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reloadLocked()
}

func (s *PhotoStore) ReloadIfChanged() (bool, error) {
	stat, err := os.Stat(s.path)
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !stat.ModTime().After(s.modTime) {
		return false, nil
	}
	if err := s.reloadLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *PhotoStore) saveLocked() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	if stat, err := os.Stat(s.path); err == nil {
		s.modTime = stat.ModTime()
	}
	return nil
}

func (s *PhotoStore) reloadLocked() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		s.data = PhotoData{Photos: make(map[string]PhotoRecord)}
		if stat, err := os.Stat(s.path); err == nil {
			s.modTime = stat.ModTime()
		}
		return nil
	}

	var data PhotoData
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	if data.Photos == nil {
		data.Photos = make(map[string]PhotoRecord)
	}
	s.data = data
	if stat, err := os.Stat(s.path); err == nil {
		s.modTime = stat.ModTime()
	}
	return nil
}

func (s *PhotoStore) nextIDLocked(now time.Time) string {
	base := strconv.FormatInt(now.UnixNano(), 36)
	id := base
	for i := 1; ; i++ {
		if _, exists := s.data.Photos[id]; !exists {
			return id
		}
		id = base + "-" + strconv.Itoa(i)
	}
}

func (s *PhotoStore) logError(msg string, err error) {
	if s.log != nil {
		s.log.Error(msg, zap.Error(err))
	}
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, raw := range tags {
		for _, part := range strings.Split(raw, ",") {
			tag := normalizeTag(part)
			if tag == "" {
				continue
			}
			if _, ok := seen[tag]; ok {
				continue
			}
			seen[tag] = struct{}{}
			out = append(out, tag)
		}
	}
	sort.Strings(out)
	return out
}

func normalizeTag(tag string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(tag)), "#")
}

func hasTag(rec PhotoRecord, tag string) bool {
	if tag == "" {
		return true
	}
	for _, candidate := range rec.Tags {
		if normalizeTag(candidate) == tag {
			return true
		}
	}
	return false
}

func localPhotoFiles(dir string, extensions []string) ([]string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, nil
	}

	stat, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, nil
	}

	allowed := localPhotoExtensionSet(extensions)
	var files []string
	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if _, ok := allowed[ext]; !ok {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

package main

import (
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"
)

type PhotoConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Path            string   `yaml:"path"`
	LocalDir        string   `yaml:"local_dir"`
	LocalExtensions []string `yaml:"local_extensions"`
	AutoReply       bool     `yaml:"auto_reply"`
	AutoReplyChance float64  `yaml:"auto_reply_chance"`
	Cooldown        string   `yaml:"cooldown"`
	Triggers        []string `yaml:"triggers"`
}

type PhotoData struct {
	Photos map[string]PhotoRecord `json:"photos"`
}

type PhotoRecord struct {
	ID           string   `json:"id"`
	FileID       string   `json:"file_id"`
	UniqueID     string   `json:"unique_id,omitempty"`
	Caption      string   `json:"caption,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	AddedByID    int64    `json:"added_by_id"`
	AddedByName  string   `json:"added_by_name"`
	AddedAtUnix  int64    `json:"added_at_unix"`
	Uses         int64    `json:"uses"`
	LastUsedUnix int64    `json:"last_used_unix"`
}

type PhotoStore struct {
	mu      sync.RWMutex
	path    string
	log     *AsyncLogger
	data    PhotoData
	modTime time.Time
	rng     *rand.Rand
}

func NewPhotoStore(path string, logger *AsyncLogger) (*PhotoStore, error) {
	s := &PhotoStore{
		path: path,
		log:  logger,
		data: PhotoData{Photos: make(map[string]PhotoRecord)},
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		if stat, err := os.Stat(path); err == nil {
			s.modTime = stat.ModTime()
		}
		return s, nil
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return nil, err
	}
	if s.data.Photos == nil {
		s.data.Photos = make(map[string]PhotoRecord)
	}
	if stat, err := os.Stat(path); err == nil {
		s.modTime = stat.ModTime()
	}
	return s, nil
}

func (s *PhotoStore) Add(photo *tele.Photo, tags []string, caption string, admin *tele.User) (PhotoRecord, error) {
	if photo == nil || photo.FileID == "" {
		return PhotoRecord{}, errors.New("photo has no Telegram file_id")
	}

	now := time.Now().UTC()
	rec := PhotoRecord{
		FileID:      photo.FileID,
		UniqueID:    photo.UniqueID,
		Caption:     strings.TrimSpace(caption),
		Tags:        normalizeTags(tags),
		AddedAtUnix: now.Unix(),
	}
	if admin != nil {
		rec.AddedByID = admin.ID
		rec.AddedByName = displayNameFromUser(admin)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data.Photos == nil {
		s.data.Photos = make(map[string]PhotoRecord)
	}
	rec.ID = s.nextIDLocked(now)
	s.data.Photos[rec.ID] = rec
	if err := s.saveLocked(); err != nil {
		return PhotoRecord{}, err
	}
	return rec, nil
}

func (s *PhotoStore) Delete(id string) (PhotoRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.data.Photos[id]
	if !ok {
		return PhotoRecord{}, false, nil
	}
	delete(s.data.Photos, id)
	if err := s.saveLocked(); err != nil {
		return PhotoRecord{}, false, err
	}
	return rec, true, nil
}

func (s *PhotoStore) List(tag string) []PhotoRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tag = normalizeTag(tag)
	records := make([]PhotoRecord, 0, len(s.data.Photos))
	for _, rec := range s.data.Photos {
		if tag == "" || hasTag(rec, tag) {
			records = append(records, rec)
		}
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].AddedAtUnix > records[j].AddedAtUnix
	})
	return records
}

func (s *PhotoStore) Random(tag string) (PhotoRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tag = normalizeTag(tag)
	candidates := make([]PhotoRecord, 0, len(s.data.Photos))
	for _, rec := range s.data.Photos {
		if tag == "" || hasTag(rec, tag) {
			candidates = append(candidates, rec)
		}
	}
	if len(candidates) == 0 {
		return PhotoRecord{}, false
	}

	rec := candidates[s.rng.Intn(len(candidates))]
	rec.Uses++
	rec.LastUsedUnix = time.Now().UTC().Unix()
	s.data.Photos[rec.ID] = rec
	if err := s.saveLocked(); err != nil {
		s.logError("photo database save failed", err)
	}
	return rec, true
}

func (s *PhotoStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data.Photos)
}

func (s *PhotoStore) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

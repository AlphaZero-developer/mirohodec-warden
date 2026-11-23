package main

import (
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

type QuoteConfig struct {
	Enabled           bool    `yaml:"enabled"`
	Path              string  `yaml:"path"`
	AutoReply         bool    `yaml:"auto_reply"`
	AutoReplyChance   float64 `yaml:"auto_reply_chance"`
	AutoReplyCooldown string  `yaml:"auto_reply_cooldown"`
	ActiveUserWindow  string  `yaml:"active_user_window"`
	PhotoChance       float64 `yaml:"photo_chance"`
}

type QuoteData struct {
	Quotes map[string]QuoteRecord `json:"quotes"`
}

type QuoteRecord struct {
	ID           string   `json:"id"`
	Text         string   `json:"text"`
	Author       string   `json:"author,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	AddedByID    int64    `json:"added_by_id"`
	AddedByName  string   `json:"added_by_name"`
	AddedAtUnix  int64    `json:"added_at_unix"`
	Uses         int64    `json:"uses"`
	LastUsedUnix int64    `json:"last_used_unix"`
}

type QuoteStore struct {
	mu      sync.RWMutex
	path    string
	log     *AsyncLogger
	data    QuoteData
	modTime time.Time
	rng     *rand.Rand
}

func NewQuoteStore(path string, logger *AsyncLogger) (*QuoteStore, error) {
	s := &QuoteStore{
		path: path,
		log:  logger,
		data: QuoteData{Quotes: make(map[string]QuoteRecord)},
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
	data, err := parseQuoteData(raw)
	if err != nil {
		return nil, err
	}
	s.data = data
	if stat, err := os.Stat(path); err == nil {
		s.modTime = stat.ModTime()
	}
	return s, nil
}

func (s *QuoteStore) Add(text, author string, tags []string, admin *tele.User) (QuoteRecord, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return QuoteRecord{}, errors.New("quote text is empty")
	}

	now := time.Now().UTC()
	rec := QuoteRecord{
		Text:        text,
		Author:      strings.TrimSpace(author),
		Tags:        normalizeTags(tags),
		AddedAtUnix: now.Unix(),
	}
	if admin != nil {
		rec.AddedByID = admin.ID
		rec.AddedByName = displayNameFromUser(admin)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data.Quotes == nil {
		s.data.Quotes = make(map[string]QuoteRecord)
	}
	rec.ID = s.nextIDLocked(now)
	s.data.Quotes[rec.ID] = rec
	if err := s.saveLocked(); err != nil {
		return QuoteRecord{}, err
	}
	return rec, nil
}

func (s *QuoteStore) Delete(id string) (QuoteRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.data.Quotes[id]
	if !ok {
		return QuoteRecord{}, false, nil
	}
	delete(s.data.Quotes, id)
	if err := s.saveLocked(); err != nil {
		return QuoteRecord{}, false, err
	}
	return rec, true, nil
}

func (s *QuoteStore) Update(id, text, author string, tags []string) (QuoteRecord, bool, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return QuoteRecord{}, false, errors.New("quote text is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.data.Quotes[id]
	if !ok {
		return QuoteRecord{}, false, nil
	}
	rec.Text = text
	rec.Author = strings.TrimSpace(author)
	rec.Tags = normalizeTags(tags)
	s.data.Quotes[id] = rec
	if err := s.saveLocked(); err != nil {
		return QuoteRecord{}, false, err
	}
	return rec, true, nil
}

func (s *QuoteStore) List(tag string) []QuoteRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tag = normalizeTag(tag)
	records := make([]QuoteRecord, 0, len(s.data.Quotes))
	for _, rec := range s.data.Quotes {
		if tag == "" || hasQuoteTag(rec, tag) {
			records = append(records, rec)
		}
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].AddedAtUnix > records[j].AddedAtUnix
	})
	return records
}

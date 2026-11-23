package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type FAQConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Path           string `yaml:"path"`
	AutoReply      bool   `yaml:"auto_reply"`
	RequireMention bool   `yaml:"require_mention"`
	EntryCooldown  string `yaml:"entry_cooldown"`
	UserCooldown   string `yaml:"user_cooldown"`
}

type FAQData struct {
	FAQs map[string]FAQRecord `json:"faqs"`
}

type FAQRecord struct {
	ID           string   `json:"id"`
	Patterns     []string `json:"patterns"`
	Answer       string   `json:"answer"`
	Enabled      bool     `json:"enabled"`
	Uses         int64    `json:"uses"`
	LastUsedUnix int64    `json:"last_used_unix"`
}

type FAQStore struct {
	mu      sync.RWMutex
	path    string
	log     *AsyncLogger
	data    FAQData
	modTime time.Time
}

func NewFAQStore(path string, logger *AsyncLogger) (*FAQStore, error) {
	s := &FAQStore{
		path: path,
		log:  logger,
		data: FAQData{FAQs: defaultFAQRecords()},
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
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
		return s, nil
	}
	data, err := parseFAQData(raw)
	if err != nil {
		return nil, err
	}
	s.data = data
	if stat, err := os.Stat(path); err == nil {
		s.modTime = stat.ModTime()
	}
	return s, nil
}

func (s *FAQStore) Add(patterns []string, answer string) (FAQRecord, error) {
	patterns = normalizeFAQPatterns(patterns)
	answer = strings.TrimSpace(answer)
	if len(patterns) == 0 {
		return FAQRecord{}, errors.New("faq patterns are empty")
	}
	if answer == "" {
		return FAQRecord{}, errors.New("faq answer is empty")
	}

	now := time.Now().UTC()
	rec := FAQRecord{
		Patterns: patterns,
		Answer:   answer,
		Enabled:  true,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data.FAQs == nil {
		s.data.FAQs = make(map[string]FAQRecord)
	}
	rec.ID = s.nextIDLocked(now)
	s.data.FAQs[rec.ID] = rec
	if err := s.saveLocked(); err != nil {
		return FAQRecord{}, err
	}
	return rec, nil
}

func (s *FAQStore) Delete(id string) (FAQRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.data.FAQs[id]
	if !ok {
		return FAQRecord{}, false, nil
	}
	delete(s.data.FAQs, id)
	if err := s.saveLocked(); err != nil {
		return FAQRecord{}, false, err
	}
	return rec, true, nil
}

func (s *FAQStore) Update(id string, patterns []string, answer string, enabled bool) (FAQRecord, bool, error) {
	patterns = normalizeFAQPatterns(patterns)
	answer = strings.TrimSpace(answer)
	if len(patterns) == 0 {
		return FAQRecord{}, false, errors.New("faq patterns are empty")
	}
	if answer == "" {
		return FAQRecord{}, false, errors.New("faq answer is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.data.FAQs[id]
	if !ok {
		return FAQRecord{}, false, nil
	}
	rec.Patterns = patterns
	rec.Answer = answer
	rec.Enabled = enabled
	s.data.FAQs[id] = rec
	if err := s.saveLocked(); err != nil {
		return FAQRecord{}, false, err
	}
	return rec, true, nil
}

func (s *FAQStore) List() []FAQRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]FAQRecord, 0, len(s.data.FAQs))
	for _, rec := range s.data.FAQs {
		records = append(records, rec)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].ID < records[j].ID
	})
	return records
}

func (s *FAQStore) Match(question string) (FAQRecord, bool) {
	normalized := normalizeFAQText(question)
	if normalized == "" {
		return FAQRecord{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var best FAQRecord
	bestScore := 0
	for _, rec := range s.data.FAQs {
		if !rec.Enabled || strings.TrimSpace(rec.Answer) == "" {
			continue
		}
		score := faqMatchScore(normalized, rec.Patterns)
		if score > bestScore {
			best = rec
			bestScore = score
		}
	}
	return best, bestScore > 0
}

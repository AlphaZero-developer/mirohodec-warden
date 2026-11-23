package main

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (s *QuoteStore) Random(tag string) (QuoteRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tag = normalizeTag(tag)
	candidates := make([]QuoteRecord, 0, len(s.data.Quotes))
	for _, rec := range s.data.Quotes {
		if tag == "" || hasQuoteTag(rec, tag) {
			candidates = append(candidates, rec)
		}
	}
	if len(candidates) == 0 {
		return QuoteRecord{}, false
	}

	rec := candidates[s.rng.Intn(len(candidates))]
	rec.Uses++
	rec.LastUsedUnix = time.Now().UTC().Unix()
	s.data.Quotes[rec.ID] = rec
	if err := s.saveLocked(); err != nil {
		s.logError("quote database save failed", err)
	}
	return rec, true
}

func (s *QuoteStore) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *QuoteStore) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reloadLocked()
}

func (s *QuoteStore) ReloadIfChanged() (bool, error) {
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

func (s *QuoteStore) saveLocked() error {
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

func (s *QuoteStore) reloadLocked() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		s.data = QuoteData{Quotes: make(map[string]QuoteRecord)}
		if stat, err := os.Stat(s.path); err == nil {
			s.modTime = stat.ModTime()
		}
		return nil
	}

	data, err := parseQuoteData(raw)
	if err != nil {
		return err
	}
	s.data = data
	if stat, err := os.Stat(s.path); err == nil {
		s.modTime = stat.ModTime()
	}
	return nil
}

func (s *QuoteStore) nextIDLocked(now time.Time) string {
	base := strconv.FormatInt(now.UnixNano(), 36)
	id := base
	for i := 1; ; i++ {
		if _, exists := s.data.Quotes[id]; !exists {
			return id
		}
		id = base + "-" + strconv.Itoa(i)
	}
}

func (s *QuoteStore) logError(msg string, err error) {
	if s.log != nil {
		s.log.Error(msg, zap.Error(err))
	}
}

func parseQuoteData(raw []byte) (QuoteData, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return QuoteData{}, err
	}
	if child, ok := root["quotes"]; ok {
		records, err := parseQuoteRecordMap(child)
		if err != nil {
			return QuoteData{}, err
		}
		return QuoteData{Quotes: records}, nil
	}
	records, err := parseQuoteRecordMap(raw)
	if err != nil {
		return QuoteData{}, err
	}
	return QuoteData{Quotes: records}, nil
}

func parseQuoteRecordMap(raw []byte) (map[string]QuoteRecord, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	records := make(map[string]QuoteRecord, len(values))
	for id, rawValue := range values {
		var text string
		if err := json.Unmarshal(rawValue, &text); err == nil {
			records[id] = QuoteRecord{ID: id, Text: strings.TrimSpace(text)}
			continue
		}
		var rec QuoteRecord
		if err := json.Unmarshal(rawValue, &rec); err != nil {
			return nil, err
		}
		if rec.ID == "" {
			rec.ID = id
		}
		rec.Text = strings.TrimSpace(rec.Text)
		rec.Author = strings.TrimSpace(rec.Author)
		rec.Tags = normalizeTags(rec.Tags)
		records[id] = rec
	}
	return records, nil
}

func hasQuoteTag(rec QuoteRecord, tag string) bool {
	for _, existing := range rec.Tags {
		if normalizeTag(existing) == tag {
			return true
		}
	}
	return false
}

package main

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (s *FAQStore) MarkUsed(id string) {
	if id == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.data.FAQs[id]
	if !ok {
		return
	}
	rec.Uses++
	rec.LastUsedUnix = time.Now().UTC().Unix()
	s.data.FAQs[id] = rec
	if err := s.saveLocked(); err != nil {
		s.logError("faq database save failed", err)
	}
}

func (s *FAQStore) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *FAQStore) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reloadLocked()
}

func (s *FAQStore) ReloadIfChanged() (bool, error) {
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

func (s *FAQStore) saveLocked() error {
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

func (s *FAQStore) reloadLocked() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		s.data = FAQData{FAQs: defaultFAQRecords()}
		if err := s.saveLocked(); err != nil {
			return err
		}
		return nil
	}

	data, err := parseFAQData(raw)
	if err != nil {
		return err
	}
	s.data = data
	if stat, err := os.Stat(s.path); err == nil {
		s.modTime = stat.ModTime()
	}
	return nil
}

func (s *FAQStore) nextIDLocked(now time.Time) string {
	base := "faq_" + strconv.FormatInt(now.UnixNano(), 36)
	id := base
	for i := 1; ; i++ {
		if _, exists := s.data.FAQs[id]; !exists {
			return id
		}
		id = base + "-" + strconv.Itoa(i)
	}
}

func (s *FAQStore) logError(msg string, err error) {
	if s.log != nil {
		s.log.Error(msg, zap.Error(err))
	}
}

func parseFAQData(raw []byte) (FAQData, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return FAQData{}, err
	}
	for _, key := range []string{"faqs", "faq"} {
		if child, ok := root[key]; ok {
			records, err := parseFAQRecordMap(child)
			if err != nil {
				return FAQData{}, err
			}
			return FAQData{FAQs: ensureFAQDefaults(records)}, nil
		}
	}
	records, err := parseFAQRecordMap(raw)
	if err != nil {
		return FAQData{}, err
	}
	return FAQData{FAQs: ensureFAQDefaults(records)}, nil
}

func parseFAQRecordMap(raw []byte) (map[string]FAQRecord, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return make(map[string]FAQRecord), nil
	}

	out := make(map[string]FAQRecord, len(values))
	for id, rawValue := range values {
		var answer string
		if err := json.Unmarshal(rawValue, &answer); err == nil {
			out[id] = FAQRecord{
				ID:       id,
				Patterns: []string{id},
				Answer:   strings.TrimSpace(answer),
				Enabled:  true,
			}
			continue
		}

		var rec FAQRecord
		if err := json.Unmarshal(rawValue, &rec); err != nil {
			return nil, err
		}
		if rec.ID == "" {
			rec.ID = id
		}
		if !strings.Contains(string(rawValue), `"enabled"`) {
			rec.Enabled = true
		}
		out[id] = rec
	}
	return out, nil
}

func ensureFAQDefaults(records map[string]FAQRecord) map[string]FAQRecord {
	if len(records) == 0 {
		return defaultFAQRecords()
	}
	for id, rec := range records {
		if rec.ID == "" {
			rec.ID = id
		}
		rec.Patterns = normalizeFAQPatterns(rec.Patterns)
		if len(rec.Patterns) == 0 {
			rec.Patterns = []string{rec.ID}
		}
		rec.Answer = strings.TrimSpace(rec.Answer)
		records[id] = rec
	}
	return records
}

package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/lib/pq"
	"go.uber.org/zap"
)

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanPhoto(row scanner) (PhotoRecord, bool) {
	var rec PhotoRecord
	err := row.Scan(&rec.ID, &rec.FileID, &rec.UniqueID, &rec.Caption, pq.Array(&rec.Tags), &rec.AddedByID, &rec.AddedByName, &rec.AddedAtUnix, &rec.Uses, &rec.LastUsedUnix)
	return rec, err == nil
}

func scanQuote(row scanner) (QuoteRecord, bool) {
	var rec QuoteRecord
	err := row.Scan(&rec.ID, &rec.Text, &rec.Author, pq.Array(&rec.Tags), &rec.AddedByID, &rec.AddedByName, &rec.AddedAtUnix, &rec.Uses, &rec.LastUsedUnix)
	return rec, err == nil
}

func scanFAQ(row scanner) (FAQRecord, bool) {
	var rec FAQRecord
	err := row.Scan(&rec.ID, pq.Array(&rec.Patterns), &rec.Answer, &rec.Enabled, &rec.Uses, &rec.LastUsedUnix)
	return rec, err == nil
}

func scanOwnerSleep(row scanner) (OwnerSleepState, bool) {
	state, ok, _ := scanOwnerSleepRow(row)
	return state, ok
}

func scanOwnerSleepRow(row scanner) (OwnerSleepState, bool, error) {
	var state OwnerSleepState
	var calls []byte
	err := row.Scan(&state.UserID, &state.Username, &state.DisplayName, pq.Array(&state.Aliases), &state.SinceUnix, &calls)
	if errors.Is(err, sql.ErrNoRows) {
		return OwnerSleepState{}, false, nil
	}
	if err != nil {
		return OwnerSleepState{}, false, err
	}
	if len(calls) > 0 {
		_ = json.Unmarshal(calls, &state.Calls)
	}
	return state, true, nil
}

func moderationKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "jail", "jailed", "rehab":
		return "jail"
	case "mute", "muted":
		return "mute"
	case "forward", "forwards", "forwardban", "forward_ban":
		return "forward"
	case "ban", "banned", "banforever":
		return "ban"
	default:
		return ""
	}
}

func (s *PostgresStore) logError(msg string, err error) {
	if s.log != nil && err != nil {
		s.log.Error(msg, zap.Error(err))
	}
}

func loadLegacyState(path string) (State, error) {
	var state State
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{Chats: make(map[string]*ChatState), ManagedChats: make(map[string]ChatSettings)}, nil
	}
	if err != nil {
		return state, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return State{Chats: make(map[string]*ChatState), ManagedChats: make(map[string]ChatSettings)}, nil
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return state, err
	}
	if state.Chats == nil {
		state.Chats = make(map[string]*ChatState)
	}
	if state.ManagedChats == nil {
		state.ManagedChats = make(map[string]ChatSettings)
	}
	return state, nil
}

func loadLegacyPhotos(path string) (PhotoData, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return PhotoData{Photos: make(map[string]PhotoRecord)}, nil
	}
	if err != nil {
		return PhotoData{}, err
	}
	var data PhotoData
	if len(strings.TrimSpace(string(raw))) == 0 {
		return PhotoData{Photos: make(map[string]PhotoRecord)}, nil
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return PhotoData{}, err
	}
	if data.Photos == nil {
		data.Photos = make(map[string]PhotoRecord)
	}
	return data, nil
}

func loadLegacyQuotes(path string) (QuoteData, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return QuoteData{Quotes: make(map[string]QuoteRecord)}, nil
	}
	if err != nil {
		return QuoteData{}, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return QuoteData{Quotes: make(map[string]QuoteRecord)}, nil
	}
	return parseQuoteData(raw)
}

func loadLegacyFAQs(path string) (FAQData, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return FAQData{FAQs: defaultFAQRecords()}, nil
	}
	if err != nil {
		return FAQData{}, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return FAQData{FAQs: defaultFAQRecords()}, nil
	}
	return parseFAQData(raw)
}

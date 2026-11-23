package main

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	tele "gopkg.in/telebot.v3"
)

func (q *PostgresQuoteStore) Add(text, author string, tags []string, admin *tele.User) (QuoteRecord, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return QuoteRecord{}, errors.New("quote text is empty")
	}
	now := time.Now().UTC()
	rec := QuoteRecord{ID: strconv.FormatInt(now.UnixNano(), 36), Text: text, Author: strings.TrimSpace(author), Tags: normalizeTags(tags), AddedAtUnix: now.Unix()}
	if admin != nil {
		rec.AddedByID = admin.ID
		rec.AddedByName = displayNameFromUser(admin)
	}
	return rec, q.base.insertQuote(rec)
}

func (s *PostgresStore) insertQuote(rec QuoteRecord) error {
	if rec.ID == "" {
		rec.ID = strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
	}
	_, err := s.db.Exec(`INSERT INTO quotes(id,text,author,tags,added_by_id,added_by_name,added_at_unix,uses,last_used_unix)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(id) DO NOTHING`,
		rec.ID, rec.Text, rec.Author, pq.Array(normalizeTags(rec.Tags)), rec.AddedByID, rec.AddedByName, rec.AddedAtUnix, rec.Uses, rec.LastUsedUnix)
	return err
}

func (q *PostgresQuoteStore) Update(id, text, author string, tags []string) (QuoteRecord, bool, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return QuoteRecord{}, false, errors.New("quote text is empty")
	}
	_, err := q.base.db.Exec(`UPDATE quotes SET text=$2, author=$3, tags=$4 WHERE id=$1`, id, text, strings.TrimSpace(author), pq.Array(normalizeTags(tags)))
	if err != nil {
		return QuoteRecord{}, false, err
	}
	rec, ok := q.base.quoteByID(id)
	return rec, ok, nil
}

func (q *PostgresQuoteStore) Delete(id string) (QuoteRecord, bool, error) {
	rec, ok := q.base.quoteByID(id)
	if !ok {
		return QuoteRecord{}, false, nil
	}
	_, err := q.base.db.Exec(`DELETE FROM quotes WHERE id=$1`, id)
	return rec, err == nil, err
}

func (q *PostgresQuoteStore) List(tag string) []QuoteRecord {
	tag = normalizeTag(tag)
	query := `SELECT id,text,author,tags,added_by_id,added_by_name,added_at_unix,uses,last_used_unix FROM quotes`
	args := []interface{}{}
	if tag != "" {
		query += ` WHERE $1=ANY(tags)`
		args = append(args, tag)
	}
	query += ` ORDER BY added_at_unix DESC`
	rows, err := q.base.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []QuoteRecord
	for rows.Next() {
		if rec, ok := scanQuote(rows); ok {
			out = append(out, rec)
		}
	}
	return out
}

func (q *PostgresQuoteStore) Random(tag string) (QuoteRecord, bool) {
	records := q.List(tag)
	if len(records) == 0 {
		return QuoteRecord{}, false
	}
	rec := records[q.base.rng.Intn(len(records))]
	rec.Uses++
	rec.LastUsedUnix = time.Now().UTC().Unix()
	_, _ = q.base.db.Exec(`UPDATE quotes SET uses=$2,last_used_unix=$3 WHERE id=$1`, rec.ID, rec.Uses, rec.LastUsedUnix)
	return rec, true
}

func (s *PostgresStore) quoteByID(id string) (QuoteRecord, bool) {
	row := s.db.QueryRow(`SELECT id,text,author,tags,added_by_id,added_by_name,added_at_unix,uses,last_used_unix FROM quotes WHERE id=$1`, id)
	return scanQuote(row)
}

func (f *PostgresFAQStore) Add(patterns []string, answer string) (FAQRecord, error) {
	patterns = normalizeFAQPatterns(patterns)
	answer = strings.TrimSpace(answer)
	if len(patterns) == 0 {
		return FAQRecord{}, errors.New("faq patterns are empty")
	}
	if answer == "" {
		return FAQRecord{}, errors.New("faq answer is empty")
	}
	now := time.Now().UTC()
	rec := FAQRecord{ID: "faq_" + strconv.FormatInt(now.UnixNano(), 36), Patterns: patterns, Answer: answer, Enabled: true}
	return rec, f.base.insertFAQ(rec)
}

func (s *PostgresStore) insertFAQ(rec FAQRecord) error {
	if rec.ID == "" {
		rec.ID = "faq_" + strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
	}
	if !rec.Enabled {

	}
	_, err := s.db.Exec(`INSERT INTO faqs(id,patterns,answer,enabled,uses,last_used_unix)
		VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(id) DO NOTHING`,
		rec.ID, pq.Array(normalizeFAQPatterns(rec.Patterns)), rec.Answer, rec.Enabled, rec.Uses, rec.LastUsedUnix)
	return err
}

func (f *PostgresFAQStore) Update(id string, patterns []string, answer string, enabled bool) (FAQRecord, bool, error) {
	patterns = normalizeFAQPatterns(patterns)
	answer = strings.TrimSpace(answer)
	if len(patterns) == 0 {
		return FAQRecord{}, false, errors.New("faq patterns are empty")
	}
	if answer == "" {
		return FAQRecord{}, false, errors.New("faq answer is empty")
	}
	_, err := f.base.db.Exec(`UPDATE faqs SET patterns=$2, answer=$3, enabled=$4 WHERE id=$1`, id, pq.Array(patterns), answer, enabled)
	if err != nil {
		return FAQRecord{}, false, err
	}
	rec, ok := f.base.faqByID(id)
	return rec, ok, nil
}

func (f *PostgresFAQStore) Delete(id string) (FAQRecord, bool, error) {
	rec, ok := f.base.faqByID(id)
	if !ok {
		return FAQRecord{}, false, nil
	}
	_, err := f.base.db.Exec(`DELETE FROM faqs WHERE id=$1`, id)
	return rec, err == nil, err
}

func (f *PostgresFAQStore) List() []FAQRecord {
	rows, err := f.base.db.Query(`SELECT id,patterns,answer,enabled,uses,last_used_unix FROM faqs ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []FAQRecord
	for rows.Next() {
		if rec, ok := scanFAQ(rows); ok {
			out = append(out, rec)
		}
	}
	return out
}

func (f *PostgresFAQStore) Match(question string) (FAQRecord, bool) {
	normalized := normalizeFAQText(question)
	if normalized == "" {
		return FAQRecord{}, false
	}
	var best FAQRecord
	bestScore := 0
	for _, rec := range f.List() {
		if !rec.Enabled || strings.TrimSpace(rec.Answer) == "" {
			continue
		}
		if score := faqMatchScore(normalized, rec.Patterns); score > bestScore {
			best = rec
			bestScore = score
		}
	}
	return best, bestScore > 0
}

func (f *PostgresFAQStore) MarkUsed(id string) {
	_, _ = f.base.db.Exec(`UPDATE faqs SET uses=uses+1, last_used_unix=$2 WHERE id=$1`, id, time.Now().UTC().Unix())
}

func (s *PostgresStore) faqByID(id string) (FAQRecord, bool) {
	row := s.db.QueryRow(`SELECT id,patterns,answer,enabled,uses,last_used_unix FROM faqs WHERE id=$1`, id)
	return scanFAQ(row)
}

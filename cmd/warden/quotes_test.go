package main

import "testing"

func TestParseQuoteDataSupportsLegacyStringMap(t *testing.T) {
	data, err := parseQuoteData([]byte(`{"quotes":{"laya_1":"Скоро."}}`))
	if err != nil {
		t.Fatalf("parseQuoteData: %v", err)
	}
	rec, ok := data.Quotes["laya_1"]
	if !ok {
		t.Fatal("missing quote")
	}
	if rec.ID != "laya_1" || rec.Text != "Скоро." {
		t.Fatalf("unexpected quote: %#v", rec)
	}
}

func TestParseQuoteDataSupportsRecordMap(t *testing.T) {
	raw := []byte(`{"quotes":{"q1":{"id":"q1","text":"Текст","author":"Лайя","tags":["faq"]}}}`)
	data, err := parseQuoteData(raw)
	if err != nil {
		t.Fatalf("parseQuoteData: %v", err)
	}
	rec := data.Quotes["q1"]
	if rec.Author != "Лайя" || len(rec.Tags) != 1 || rec.Tags[0] != "faq" {
		t.Fatalf("unexpected quote record: %#v", rec)
	}
}

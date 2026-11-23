CREATE INDEX IF NOT EXISTS karma_events_lookup_idx ON karma_events(chat_id, user_id, reason, day, by_user_id, created_unix)

CREATE INDEX IF NOT EXISTS warnings_user_idx ON warnings(chat_id, user_id, active, expires_unix)

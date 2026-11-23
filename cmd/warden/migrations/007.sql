CREATE INDEX IF NOT EXISTS members_username_idx ON members(chat_id, lower(username))

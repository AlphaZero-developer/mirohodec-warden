ALTER TABLE managed_chats ADD COLUMN IF NOT EXISTS karma_manual_cooldown TEXT NOT NULL DEFAULT ''

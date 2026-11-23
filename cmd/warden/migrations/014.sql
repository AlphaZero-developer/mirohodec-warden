CREATE TABLE IF NOT EXISTS karma_totals (
			chat_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			username TEXT NOT NULL DEFAULT '',
			score BIGINT NOT NULL DEFAULT 0,
			updated_unix BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (chat_id, user_id)
		)

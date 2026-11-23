CREATE TABLE IF NOT EXISTS warnings (
			id TEXT PRIMARY KEY,
			chat_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			display_name TEXT NOT NULL DEFAULT '',
			reason TEXT NOT NULL DEFAULT '',
			by_admin_id BIGINT NOT NULL DEFAULT 0,
			by_admin_name TEXT NOT NULL DEFAULT '',
			created_unix BIGINT NOT NULL DEFAULT 0,
			expires_unix BIGINT NOT NULL DEFAULT 0,
			active BOOLEAN NOT NULL DEFAULT TRUE
		)

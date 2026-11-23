CREATE TABLE IF NOT EXISTS karma_events (
			id BIGSERIAL PRIMARY KEY,
			chat_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			delta BIGINT NOT NULL,
			reason TEXT NOT NULL,
			by_user_id BIGINT NOT NULL DEFAULT 0,
			by_user_name TEXT NOT NULL DEFAULT '',
			message_id INTEGER NOT NULL DEFAULT 0,
			day TEXT NOT NULL DEFAULT '',
			created_unix BIGINT NOT NULL DEFAULT 0
		)

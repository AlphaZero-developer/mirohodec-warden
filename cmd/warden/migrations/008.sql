CREATE TABLE IF NOT EXISTS moderation_records (
			chat_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			kind TEXT NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			display_name TEXT NOT NULL DEFAULT '',
			reason TEXT NOT NULL DEFAULT '',
			rehab_ward TEXT NOT NULL DEFAULT '',
			by_admin_id BIGINT NOT NULL DEFAULT 0,
			by_admin_name TEXT NOT NULL DEFAULT '',
			since_unix BIGINT NOT NULL DEFAULT 0,
			started_at BIGINT NOT NULL DEFAULT 0,
			until_unix BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (chat_id, user_id, kind)
		)

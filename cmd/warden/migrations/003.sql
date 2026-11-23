CREATE TABLE IF NOT EXISTS owner_sleep (
			user_id BIGINT PRIMARY KEY,
			username TEXT NOT NULL DEFAULT '',
			display_name TEXT NOT NULL DEFAULT '',
			aliases TEXT[] NOT NULL DEFAULT '{}',
			since_unix BIGINT NOT NULL DEFAULT 0,
			calls JSONB NOT NULL DEFAULT '[]'
		)

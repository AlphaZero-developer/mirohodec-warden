CREATE TABLE IF NOT EXISTS photos (
			id TEXT PRIMARY KEY,
			file_id TEXT NOT NULL,
			unique_id TEXT NOT NULL DEFAULT '',
			caption TEXT NOT NULL DEFAULT '',
			tags TEXT[] NOT NULL DEFAULT '{}',
			added_by_id BIGINT NOT NULL DEFAULT 0,
			added_by_name TEXT NOT NULL DEFAULT '',
			added_at_unix BIGINT NOT NULL DEFAULT 0,
			uses BIGINT NOT NULL DEFAULT 0,
			last_used_unix BIGINT NOT NULL DEFAULT 0
		)

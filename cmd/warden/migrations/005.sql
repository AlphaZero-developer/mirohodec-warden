CREATE TABLE IF NOT EXISTS discussion_replies (
			chat_id BIGINT NOT NULL,
			topic_id INTEGER NOT NULL,
			posted_unix BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (chat_id, topic_id)
		)

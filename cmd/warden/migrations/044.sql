UPDATE moderation_records SET started_at=CASE WHEN since_unix<>0 THEN since_unix ELSE EXTRACT(EPOCH FROM NOW())::BIGINT END WHERE kind='jail' AND started_at=0

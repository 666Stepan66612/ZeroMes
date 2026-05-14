CREATE TABLE IF NOT EXISTS chats (
    id TEXT PRIMARY KEY,
    user_id UUID NOT NULL,
    companion_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, companion_id)
);

CREATE INDEX idx_chats_user_id ON chats (user_id, last_message_at DESC);
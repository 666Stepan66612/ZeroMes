ALTER TABLE chats 
ADD COLUMN encrypted_key TEXT,
ADD COLUMN key_iv VARCHAR(24);

CREATE INDEX idx_chats_encrypted_key ON chats(user_id) 
WHERE encrypted_key IS NOT NULL;
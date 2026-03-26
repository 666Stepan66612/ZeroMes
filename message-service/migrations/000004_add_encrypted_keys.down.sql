DROP INDEX IF EXISTS idx_chats_encrypted_key;

ALTER TABLE chats 
DROP COLUMN IF EXISTS encrypted_key,
DROP COLUMN IF EXISTS key_iv;
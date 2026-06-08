DROP TRIGGER IF EXISTS trigger_mark_group_for_rotation ON group_members;
DROP FUNCTION IF EXISTS mark_group_for_rotation();

ALTER TABLE messages DROP CONSTRAINT IF EXISTS chk_message_recipient;
ALTER TABLE messages DROP COLUMN IF EXISTS key_version;
ALTER TABLE messages DROP COLUMN IF EXISTS group_id;

DROP INDEX IF EXISTS idx_messages_key_version;
DROP INDEX IF EXISTS idx_messages_group_id;

DROP TABLE IF EXISTS user_group_keys;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS group_chats;

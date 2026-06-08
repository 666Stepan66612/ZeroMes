-- Group chats
CREATE TABLE IF NOT EXISTS group_chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    avatar_url TEXT,
    created_by UUID NOT NULL,
    key_version INT NOT NULL DEFAULT 0,
    needs_rotation BOOLEAN NOT NULL DEFAULT FALSE,
    rotation_in_progress BOOLEAN NOT NULL DEFAULT FALSE,
    rotation_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_group_chats_created_by ON group_chats(created_by);
CREATE INDEX idx_group_chats_needs_rotation ON group_chats(needs_rotation) WHERE needs_rotation = TRUE;

-- Group members
CREATE TABLE IF NOT EXISTS group_members (
    group_id UUID NOT NULL REFERENCES group_chats(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'member')),
    can_read_from_message_id UUID,
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    left_at TIMESTAMP,
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX idx_group_members_user_id ON group_members(user_id);
CREATE INDEX idx_group_members_active ON group_members(group_id, user_id) WHERE left_at IS NULL;

-- Encrypted group key seeds for each user
CREATE TABLE IF NOT EXISTS user_group_keys (
    user_id UUID NOT NULL,
    group_id UUID NOT NULL REFERENCES group_chats(id) ON DELETE CASCADE,
    encrypted_seed TEXT NOT NULL,
    encrypted_by UUID NOT NULL, -- who encrypted this seed (for ECDH)
    key_version INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, group_id)
);

CREATE INDEX idx_user_group_keys_group_id ON user_group_keys(group_id);

-- Alter messages table to support groups
ALTER TABLE messages
    ADD COLUMN group_id UUID REFERENCES group_chats(id) ON DELETE CASCADE,
    ADD COLUMN key_version INT DEFAULT 0,
    ALTER COLUMN recipient_id DROP NOT NULL;

-- Add constraint: either recipient_id (1-1 chat) or group_id (group chat) must be set
ALTER TABLE messages
    ADD CONSTRAINT chk_message_recipient
    CHECK (
        (recipient_id IS NOT NULL AND group_id IS NULL) OR
        (recipient_id IS NULL AND group_id IS NOT NULL)
    );

CREATE INDEX idx_messages_group_id ON messages(group_id, created_at DESC);
CREATE INDEX idx_messages_key_version ON messages(group_id, key_version);

-- Trigger: mark group for key rotation when member leaves
CREATE OR REPLACE FUNCTION mark_group_for_rotation()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.left_at IS NOT NULL AND OLD.left_at IS NULL THEN
        UPDATE group_chats
        SET needs_rotation = TRUE
        WHERE id = NEW.group_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_mark_group_for_rotation
AFTER UPDATE ON group_members
FOR EACH ROW
EXECUTE FUNCTION mark_group_for_rotation();

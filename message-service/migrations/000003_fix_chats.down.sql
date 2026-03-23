ALTER TABLE chats DROP CONSTRAINT chats_pkey;
ALTER TABLE chats ADD COLUMN id TEXT;
UPDATE chats SET id = user_id || ':' || companion_id;
ALTER TABLE chats ADD PRIMARY KEY (id);
ALTER TABLE chats ADD CONSTRAINT chats_user_id_companion_id_key UNIQUE (user_id, companion_id);
ALTER TABLE chats DROP COLUMN id;
ALTER TABLE chats ADD PRIMARY KEY (user_id, companion_id);
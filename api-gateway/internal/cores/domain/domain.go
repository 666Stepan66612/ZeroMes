package domain

type WSRequest struct {
    Type          string `json:"type"`
    ChatID        string `json:"chat_id"`
    RecipientID   string `json:"recipient_id"`
    Content       string `json:"content"`
    MessageType   string `json:"message_type"`
    MessageID     string `json:"message_id"`
    LastMessageID string `json:"last_message_id"`
    Limit         int32  `json:"limit"`
    NewContent    string `json:"new_content"`
    CompanionID   string `json:"companion_id"`
    EncryptedKey  string `json:"encrypted_key"`
    KeyIV         string `json:"key_iv"` 
}

type Message struct {
    ID        string `json:"id"`
    ChatID    string `json:"chat_id"`
    SenderID  string `json:"sender_id"`
    Content   string `json:"content"`
    CreatedAt string `json:"created_at"`
    Status    int32  `json:"status"`
}

type GetMessagesResponse struct {
    Messages []*Message  `json:"messages"`
	NextMessageId string `json:"next_message_id"`
    HasMore  bool        `json:"has_more"`
}

type WSResponse struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload,omitempty"`
}

type Chat struct {
    ID            string `json:"id"`
    CompanionID   string `json:"companion_id"`
    LastMessageAt string `json:"last_message_at"`
    EncryptedKey  string `json:"encrypted_key"`
    KeyIV         string `json:"key_iv"`  
}

type GetChatsResponse struct {
    Chats []*Chat `json:"chats"`
}

type ChatKeyUpdate struct {
    CompanionID  string `json:"companion_id"`
    EncryptedKey string `json:"encrypted_key"`
    KeyIV        string `json:"key_iv"`
}

type ChangePasswordRequest struct {
    Login       string          `json:"login"`
    OldAuthHash string          `json:"old_auth_hash"`
    NewAuthHash string          `json:"new_auth_hash"`
    NewPublicKey string         `json:"new_public_key"`
    ChatKeys    []ChatKeyUpdate `json:"chat_keys"`
}

type ChangePasswordResponse struct {
    Success      bool   `json:"success"`
    UpdatedChats int    `json:"updated_chats"`
    Message      string `json:"message"`
}

type contextKey string
const AccessTokenKey contextKey = "access_token"
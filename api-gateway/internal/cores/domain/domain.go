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
}

type Message struct {
    ID        string `json:"id"`
    ChatID    string `json:"chat_id"`
    SenderID  string `json:"sender_id"`
    Content   string `json:"content"`
    CreatedAt string `json:"created_at"`
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
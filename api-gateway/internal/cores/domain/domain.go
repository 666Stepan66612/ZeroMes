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
package domain

type Message struct {
    MessageID   string
    SenderID    string
    RecipientID string
    Content     string
    Timestamp   int64
}
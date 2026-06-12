package domain

type Message struct {
	MessageID   string
	ChatID      string
	SenderID    string
	RecipientID string
	Content     string
	Timestamp   string
	GroupID     string
	KeyVersion  int32
}

package service

import "time"

type Message struct {
	ID               string
	ChatID           string
	SenderID         string
	RecipientID      string
	EncryptedContent string
	MessageType      string
	CreatedAt        time.Time
	Status           MessageStatus
}

type ChatsList struct {
	ChatID         string
	UserID         string
	CompanionID    string
	CompanionLogin string
	CreatedAt      time.Time
	LastMessageAt  time.Time
	EncryptedKey   string
	KeyIV          string
	LastMessage    string // Encrypted content of last message
}

type MessageStatus int8

const (
	MessageStatusSent MessageStatus = iota
	MessageStatusDelivered
	MessageStatusRead
)

type ChatKeyUpdate struct {
	CompanionID  string
	EncryptedKey string
	KeyIV        string
}

type OutboxEvent struct {
	ID          string
	EventType   string
	AggregateID string
	Payload     []byte // JSON
	CreatedAt   time.Time
	ProcessedAt *time.Time
	RetryCount  int
	LastError   string
	Status      string // "pending", "processing", "completed", "failed"
}

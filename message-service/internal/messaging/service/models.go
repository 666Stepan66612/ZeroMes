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

type MessageStatus int8

const (
    MessageStatusSent MessageStatus = iota
    MessageStatusDelivered
    MessageStatusRead
)

package service

import "context"

type MessageService interface {
	SendMessage(ctx context.Context, chatID, senderID, recipientID, content, msgType string) (*Message, error)
	GetMessages(ctx context.Context, chatID, userID string, limit int, lastMessageID string) ([]*Message, error)
	MarkAsRead(ctx context.Context, chatID, userID, lastMessageID string) error
	AlterMessage(ctx context.Context, messageID, userID, newContent string) error
	DeleteMessage(ctx context.Context, messageID, userID string) error
	GetChats(ctx context.Context, userID string) ([]*ChatsList, error)
	SaveChatKeys(ctx context.Context, userID, companionID, encryptedKey, keyIV string) error
	UpdateChatKeys(ctx context.Context, userID string, keys []ChatKeyUpdate) (int, error)
	CreateGroup(ctx context.Context, name, createdBy string, memberIDs []string, seedDistributions []SeedDistribution) (*GroupChat, error)
    AddGroupMember(ctx context.Context, groupID, userID, addedBy, encryptedSeed string) error
    RemoveGroupMember(ctx context.Context, groupID, userID, removedBy string) (int, error)
    LeaveGroup(ctx context.Context, groupID, userID string) error
    GetGroupChats(ctx context.Context, userID string) ([]*GroupChat, error)
    GetGroupMembers(ctx context.Context, groupID string) ([]*GroupMember, error)
    SaveGroupKeySeed(ctx context.Context, userID, groupID, encryptedSeed, encryptedBy string, keyVersion int) error
    GetGroupKeySeed(ctx context.Context, userID, groupID string) (*GroupKeySeed, int, error)
}

type MessageRepository interface {
	CreateWithChats(ctx context.Context, msg *Message) error
	GetByChatID(ctx context.Context, chatID string, limit int, lastMessageID string) ([]*Message, error)
	GetByID(ctx context.Context, messageID string) (*Message, error)
	Delete(ctx context.Context, messageID string) error
	Alter(ctx context.Context, messageID, newContent string) error
	UpdateStatusBatch(ctx context.Context, chatID, userID, lastMessageID string, status MessageStatus) error
	GetChats(ctx context.Context, userID string) ([]*ChatsList, error)
	SaveChatKeys(ctx context.Context, userID, companionID, encryptedKey, keyIV string) error
	UpdateChatKeys(ctx context.Context, userID string, keys []ChatKeyUpdate) (int, error)
	CreateGroup(ctx context.Context, name, avatarURL, createdBy string, keyVersion int) (string, error)
    GetGroupByID(ctx context.Context, groupID string) (*GroupChat, error)
    GetGroupChats(ctx context.Context, userID string) ([]*GroupChat, error)
    AddGroupMember(ctx context.Context, groupID, userID, role string, canReadFromMessageID *string) error
    RemoveGroupMember(ctx context.Context, groupID, userID string) error
    GetGroupMembers(ctx context.Context, groupID string) ([]*GroupMember, error)
    CheckGroupMembership(ctx context.Context, groupID, userID string) (bool, string, error)
    GetActiveGroupMemberIDs(ctx context.Context, groupID string) ([]string, error)
    SaveGroupKeySeed(ctx context.Context, userID, groupID, encryptedSeed, encryptedBy string, keyVersion int) error
    GetGroupKeySeed(ctx context.Context, userID, groupID string) (*GroupKeySeed, error)
    GetCurrentKeyVersion(ctx context.Context, groupID string) (int, error)
    TryAcquireRotationLock(ctx context.Context, groupID, userID string) (bool, error)
    IncrementKeyVersion(ctx context.Context, groupID string) (int, error)
    ReleaseRotationLock(ctx context.Context, groupID string) error
    GetGroupMessages(ctx context.Context, groupID, userID string, limit int, lastMessageID string) ([]*Message, error)
    CreateGroupMessage(ctx context.Context, msg *Message) error
}

type KafkaProducer interface {
	PublishMessageSent(ctx context.Context, msg *Message) error
	PublishMessageAltered(ctx context.Context, msg *Message, newContent string) error
	PublishMessageDeleted(ctx context.Context, msg *Message) error
	PublishMessageRead(ctx context.Context, chatID, readerID, senderID, lastMessageID string) error
	Close() error
}

type OutboxRepository interface {
	SaveToOutbox(ctx context.Context, event *OutboxEvent) error
	GetPendingEvents(ctx context.Context, limit int) ([]*OutboxEvent, error)
	MarkEventProcessed(ctx context.Context, eventID string) error
	MarkEventFailed(ctx context.Context, eventID string, errorMsg string) error
	IncrementRetryCount(ctx context.Context, eventID string) error
}

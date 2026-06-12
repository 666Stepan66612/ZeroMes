package domain

type WSRequest struct {
	Type              string              `json:"type"`
	ChatID            string              `json:"chat_id"`
	RecipientID       string              `json:"recipient_id"`
	Content           string              `json:"content"`
	MessageType       string              `json:"message_type"`
	MessageID         string              `json:"message_id"`
	LastMessageID     string              `json:"last_message_id"`
	Limit             int32               `json:"limit"`
	NewContent        string              `json:"new_content"`
	CompanionID       string              `json:"companion_id"`
	EncryptedKey      string              `json:"encrypted_key"`
	KeyIV             string              `json:"key_iv"`
	UserID            string              `json:"user_id"`
	GroupID           string              `json:"group_id"`
	KeyVersion        int32               `json:"key_version"`
	GroupName         string              `json:"group_name"`
	MemberIDs         []string            `json:"member_ids"`
	AddedBy           string              `json:"added_by"`
	RemovedBy         string              `json:"removed_by"`
	EncryptedSeed     string              `json:"encrypted_seed"`
	SeedDistributions []SeedDistribution  `json:"seed_distributions"`
}

type SeedDistribution struct {
	UserID        string `json:"user_id"`
	EncryptedSeed string `json:"encrypted_seed"`
	EncryptedBy   string `json:"encrypted_by"`
}

type Message struct {
	ID               string `json:"id"`
	ChatID           string `json:"chat_id"`
	SenderID         string `json:"sender_id"`
	EncryptedContent string `json:"encrypted_content"`
	CreatedAt        string `json:"created_at"`
	Status           int32  `json:"status"`
	GroupID          string `json:"group_id,omitempty"`
	KeyVersion       int32  `json:"key_version,omitempty"`
}

type GetMessagesResponse struct {
	Messages      []*Message `json:"messages"`
	NextMessageId string     `json:"next_message_id"`
	HasMore       bool       `json:"has_more"`
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
	LastMessage   string `json:"last_message,omitempty"`
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
	Login        string          `json:"login"`
	OldAuthHash  string          `json:"old_auth_hash"`
	NewAuthHash  string          `json:"new_auth_hash"`
	NewPublicKey string          `json:"new_public_key"`
	ChatKeys     []ChatKeyUpdate `json:"chat_keys"`
}

type ChangePasswordResponse struct {
	Success      bool   `json:"success"`
	UpdatedChats int    `json:"updated_chats"`
	Message      string `json:"message"`
}

type GroupChat struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	CreatedBy  string `json:"created_by"`
	KeyVersion int32  `json:"key_version"`
	CreatedAt  string `json:"created_at"`
}

type GroupMember struct {
	UserID               string  `json:"user_id"`
	Role                 string  `json:"role"`
	JoinedAt             string  `json:"joined_at"`
	CanReadFromMessageID *string `json:"can_read_from_message_id,omitempty"`
}

type GroupKeySeed struct {
	EncryptedSeed string `json:"encrypted_seed"`
	EncryptedBy   string `json:"encrypted_by"`
	KeyVersion    int32  `json:"key_version"`
}

type GetGroupChatsResponse struct {
	Groups []*GroupChat `json:"groups"`
}

type GetGroupMembersResponse struct {
	Members []*GroupMember `json:"members"`
}

type GetGroupKeySeedResponse struct {
	Seed              *GroupKeySeed `json:"seed"`
	CurrentKeyVersion int32         `json:"current_key_version"`
}

type contextKey string

const AccessTokenKey contextKey = "access_token"

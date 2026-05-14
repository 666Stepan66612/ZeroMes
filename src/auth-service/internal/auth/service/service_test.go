package service

import (
	"auth-service/internal/cores/errors"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockUserRepository) GetByLogin(ctx context.Context, login string) (*User, error) {
	args := m.Called(ctx, login)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockUserRepository) SearchUsers(ctx context.Context, login string) ([]*UserPublic, error) {
	args := m.Called(ctx, login)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*UserPublic), args.Error(1)
}

func (m *MockUserRepository) UpdateAuthHashAndPublicKey(ctx context.Context, userID, newAuthHash, newPublicKey string) error {
	args := m.Called(ctx, userID, newAuthHash, newPublicKey)
	return args.Error(0)
}

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateTokenPair(userID string) (*TokenPair, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TokenPair), args.Error(1)
}

func (m *MockTokenService) ValidateAccessToken(token string) (string, error) {
	args := m.Called(token)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) ValidateRefreshToken(token string) (string, error) {
	args := m.Called(token)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) InvalidateRefreshToken(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockTokenService) InvalidateAccessToken(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func TestRegister_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	login := "testuser"
	authHash := "client-auth-hash-123"
	publicKey := "public-key-data"

	mockRepo.On("GetByLogin", ctx, login).Return(nil, errors.ErrUserNotFound)

	mockRepo.On("Create", ctx, mock.AnythingOfType("*service.User")).Return(nil)

	expectedTokens := &TokenPair{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
	}
	mockTokenSvc.On("GenerateTokenPair", mock.AnythingOfType("string")).Return(expectedTokens, nil)

	userPublic, tokens, err := service.Register(ctx, login, authHash, publicKey)

	assert.NoError(t, err)
	assert.NotNil(t, userPublic)
	assert.Equal(t, login, userPublic.Login)
	assert.Equal(t, publicKey, userPublic.PublicKey)
	assert.NotEmpty(t, userPublic.ID)
	assert.NotNil(t, tokens)
	assert.Equal(t, expectedTokens.AccessToken, tokens.AccessToken)
	assert.Equal(t, expectedTokens.RefreshToken, tokens.RefreshToken)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertExpectations(t)
}

func TestRegister_UserAlreadyExists(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	login := "existinguser"

	existingUser := &User{
		ID:    "existing-user-id",
		Login: login,
	}
	mockRepo.On("GetByLogin", ctx, login).Return(existingUser, nil)

	userPublic, tokens, err := service.Register(ctx, login, "hash", "key")

	assert.Error(t, err)
	assert.Equal(t, errors.ErrUserAlreadyExists, err)
	assert.Nil(t, userPublic)
	assert.Nil(t, tokens)

	mockRepo.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	login := "testuser"
	clientAuthHash := "client-auth-hash-123"

	serverSalt, _ := GenerateServerSalt()
	storedHash, _ := HashAuthHash(clientAuthHash, serverSalt)

	existingUser := &User{
		ID:         "existing-user-id",
		Login:      login,
		AuthHash:   storedHash,
		ServerSalt: serverSalt,
		PublicKey:  "public-key-data",
	}
	mockRepo.On("GetByLogin", ctx, login).Return(existingUser, nil)

	expectedTokens := &TokenPair{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
	}
	mockTokenSvc.On("GenerateTokenPair", existingUser.ID).Return(expectedTokens, nil)

	userPublic, tokens, err := service.Login(ctx, login, clientAuthHash)

	assert.NoError(t, err)
	assert.NotNil(t, userPublic)
	assert.Equal(t, login, userPublic.Login)
	assert.Equal(t, existingUser.ID, userPublic.ID)
	assert.NotNil(t, tokens)
	assert.Equal(t, expectedTokens.AccessToken, tokens.AccessToken)
	assert.Equal(t, expectedTokens.RefreshToken, tokens.RefreshToken)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertExpectations(t)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	login := "testuser"

	mockRepo.On("GetByLogin", ctx, login).Return(nil, errors.ErrUserNotFound)

	userPublic, tokens, err := service.Login(ctx, login, "wrong-hash")

	assert.Error(t, err)
	assert.Equal(t, errors.ErrInvalidCredentials, err)
	assert.Nil(t, userPublic)
	assert.Nil(t, tokens)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertNotCalled(t, "GenerateTokenPair")
}

func TestLogin_WrongPassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	login := "testuser"
	correctAuthHash := "correct-hash"
	wrongAuthHash := "wrong-hash"

	serverSalt, _ := GenerateServerSalt()
	storedHash, _ := HashAuthHash(correctAuthHash, serverSalt)

	existingUser := &User{
		ID:         "user-id",
		Login:      login,
		AuthHash:   storedHash,
		ServerSalt: serverSalt,
	}
	mockRepo.On("GetByLogin", ctx, login).Return(existingUser, nil)

	userPublic, tokens, err := service.Login(ctx, login, wrongAuthHash)

	assert.Error(t, err)
	assert.Equal(t, errors.ErrInvalidCredentials, err)
	assert.Nil(t, userPublic)
	assert.Nil(t, tokens)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertNotCalled(t, "GenerateTokenPair")
}

func TestRefreshToken_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	refreshToken := "refresh-token-test"
	userID := "test-user-id-123"
	tokenPairExpected := &TokenPair{
		AccessToken:  "test-access-token-123",
		RefreshToken: refreshToken,
	}

	mockTokenSvc.On("ValidateRefreshToken", refreshToken).Return(userID, nil)
	mockTokenSvc.On("GenerateTokenPair", userID).Return(tokenPairExpected, nil)

	tokenPair, err := service.RefreshToken(ctx, refreshToken)

	assert.NoError(t, err)
	assert.Equal(t, tokenPairExpected.AccessToken, tokenPair.AccessToken)
	assert.Equal(t, tokenPairExpected.RefreshToken, tokenPair.RefreshToken)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertExpectations(t)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	refreshToken := "refresh-token-test"

	mockTokenSvc.On("ValidateRefreshToken", refreshToken).Return("", errors.ErrInvalidToken)

	tokenPair, err := service.RefreshToken(ctx, refreshToken)

	assert.Nil(t, tokenPair)
	assert.Nil(t, tokenPair)
	assert.Error(t, err)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertNotCalled(t, "GenerateTokenPair")
}

func TestLogout_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	refreshToken := "refresh-token-test"
	expectingAccessToken := "access-token-test"

	mockTokenSvc.On("InvalidateRefreshToken", refreshToken).Return(nil)
	mockTokenSvc.On("InvalidateAccessToken", expectingAccessToken).Return(nil)

	err := service.Logout(ctx, refreshToken, expectingAccessToken)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertExpectations(t)
}

func TestLogout_Wrong(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	refreshToken := "refresh-token-test"
	expectingAccessToken := "access-token-test"

	mockTokenSvc.On("InvalidateRefreshToken", refreshToken).Return(errors.ErrInvalidToken)

	err := service.Logout(ctx, refreshToken, expectingAccessToken)
	assert.Error(t, err)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertNotCalled(t, "InvalidateAccessToken")
}

func TestSearch_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	login := "test-login"
	ti, _ := time.Parse("2006-01-02 15:04:05", "2024-05-06 12:00:00")

	expectingUsers := []*UserPublic{
		{ID: "123", Login: "test-login-1", PublicKey: "228", CreatedAt: ti},
		{ID: "456", Login: "test-login-2", PublicKey: "1337", CreatedAt: ti},
	}

	mockRepo.On("SearchUsers", ctx, login).Return(expectingUsers, nil)

	users, err := service.Search(ctx, login)
	assert.NoError(t, err)
	assert.Equal(t, expectingUsers, users)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertExpectations(t)
}

func TestSearch_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	login := "test-login"

	expectingUsers := []*UserPublic{}

	mockRepo.On("SearchUsers", ctx, login).Return(expectingUsers, nil)

	users, err := service.Search(ctx, login)
	assert.NoError(t, err)
	assert.Equal(t, expectingUsers, users)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertExpectations(t)
}

func TestSearch_Wrong(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	login := "test-login"

	mockRepo.On("SearchUsers", ctx, login).Return(nil, errors.ErrInternalServer)

	users, err := service.Search(ctx, login)

	assert.Error(t, err)
	assert.Equal(t, errors.ErrInternalServer, err)
	assert.Nil(t, users)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertExpectations(t)
}

func TestChangePassword_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	login := "test-login"
	oldAuthHash := "test-old-auth-hash"
	newAuthHash := "test-new-auth-hash"
	newPublicKey := "test-new-public-key"
	serverSalt := "123"
	hashedOldAuthHash, _ := HashAuthHash(oldAuthHash, serverSalt)
	expectingUser := &User{
		AuthHash:   hashedOldAuthHash,
		ServerSalt: serverSalt,
	}

	mockRepo.On("GetByLogin", ctx, login).Return(expectingUser, nil)

	mockRepo.On("UpdateAuthHashAndPublicKey", ctx, expectingUser.ID, mock.AnythingOfType("string"), newPublicKey).Return(nil)

	mockTokenSvc.On("InvalidateAccessToken", expectingUser.ID).Return(nil)
	mockTokenSvc.On("InvalidateRefreshToken", expectingUser.ID).Return(nil)

	userID, err := service.ChangePassword(ctx, login, oldAuthHash, newAuthHash, newPublicKey)

	assert.Equal(t, expectingUser.ID, userID)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertExpectations(t)
}
func TestChangePassword_EmptyLogin(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()

	userID, err := service.ChangePassword(ctx, "", "old-hash", "new-hash", "new-key")

	assert.Error(t, err)
	assert.Equal(t, errors.ErrInvalidInput, err)
	assert.Empty(t, userID)

	mockRepo.AssertNotCalled(t, "GetByLogin")
	mockTokenSvc.AssertNotCalled(t, "InvalidateAccessToken")
	mockTokenSvc.AssertNotCalled(t, "InvalidateRefreshToken")
}

func TestChangePassword_EmptyOldAuthHash(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()

	userID, err := service.ChangePassword(ctx, "test-login", "", "new-hash", "new-key")

	assert.Error(t, err)
	assert.Equal(t, errors.ErrInvalidInput, err)
	assert.Empty(t, userID)

	mockRepo.AssertNotCalled(t, "GetByLogin")
}

func TestChangePassword_EmptyNewAuthHash(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()

	userID, err := service.ChangePassword(ctx, "test-login", "old-hash", "", "new-key")

	assert.Error(t, err)
	assert.Equal(t, errors.ErrInvalidInput, err)
	assert.Empty(t, userID)

	mockRepo.AssertNotCalled(t, "GetByLogin")
}

func TestChangePassword_UserNotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	login := "nonexistent-user"

	mockRepo.On("GetByLogin", ctx, login).Return(nil, errors.ErrUserNotFound)

	userID, err := service.ChangePassword(ctx, login, "old-hash", "new-hash", "new-key")

	assert.Error(t, err)
	assert.Equal(t, errors.ErrUserNotFound, err)
	assert.Empty(t, userID)

	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "UpdateAuthHashAndPublicKey")
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	login := "test-login"
	correctOldHash := "correct-old-hash"
	wrongOldHash := "wrong-old-hash"

	serverSalt := "test-salt"
	hashedCorrectOld, _ := HashAuthHash(correctOldHash, serverSalt)

	existingUser := &User{
		ID:         "user-123",
		Login:      login,
		AuthHash:   hashedCorrectOld,
		ServerSalt: serverSalt,
	}

	mockRepo.On("GetByLogin", ctx, login).Return(existingUser, nil)

	userID, err := service.ChangePassword(ctx, login, wrongOldHash, "new-hash", "new-key")

	assert.Error(t, err)
	assert.Equal(t, errors.ErrInvalidOldPassword, err)
	assert.Empty(t, userID)

	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "UpdateAuthHashAndPublicKey")
}

func TestChangePassword_UpdateFailed(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenSvc := new(MockTokenService)
	service := NewAuthService(mockRepo, mockTokenSvc)

	ctx := context.Background()
	login := "test-login"
	oldAuthHash := "old-hash"
	newAuthHash := "new-hash"
	newPublicKey := "new-key"

	serverSalt := "test-salt"
	hashedOldAuthHash, _ := HashAuthHash(oldAuthHash, serverSalt)

	existingUser := &User{
		ID:         "user-123",
		Login:      login,
		AuthHash:   hashedOldAuthHash,
		ServerSalt: serverSalt,
	}

	mockRepo.On("GetByLogin", ctx, login).Return(existingUser, nil)
	mockRepo.On("UpdateAuthHashAndPublicKey", ctx, existingUser.ID, mock.AnythingOfType("string"), newPublicKey).
		Return(errors.ErrInternalServer)

	userID, err := service.ChangePassword(ctx, login, oldAuthHash, newAuthHash, newPublicKey)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update password")
	assert.Empty(t, userID)

	mockRepo.AssertExpectations(t)
	mockTokenSvc.AssertNotCalled(t, "InvalidateAccessToken")
	mockTokenSvc.AssertNotCalled(t, "InvalidateRefreshToken")
}

# Testing Guide

## Overview

ZeroMes has comprehensive unit test coverage for critical components. Tests are automatically run in CI on every push and pull request.

## Test Structure

### Backend (Go)

```
service/
├── service.go           # Business logic
├── service_test.go      # Unit tests
├── models.go            # Data models
└── interface.go         # Interfaces for mocking
```

### Test Files

- **message-service/internal/messaging/service/**
  - `service_test.go` - Service layer tests (SendMessage, GetMessages, DeleteMessage, AlterMessage, MarkAsRead, GetChats, UpdateChatKeys)
  - `slog_test.go` - Logging tests (Kafka failure scenarios, log level checks)

- **message-service/internal/cores/outbox-worker/**
  - `worker_test.go` - Outbox worker tests (retry logic, event processing, error handling)

- **auth-service/internal/auth/service/**
  - `service_test.go` - Auth service tests (Register, Login, ChangePassword, RefreshToken, Logout, Search)

- **pkg/jwt/**
  - `token_service_test.go` - JWT validation tests (token validation, expiration, algorithm confusion protection)

## Running Tests

### All Tests

```bash
# Run all backend tests
go test ./... -v

# With race detector
go test ./... -v -race

# With coverage
go test ./... -v -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Specific Service

```bash
# Message service
cd message-service
go test ./... -v

# Auth service
cd auth-service
go test ./... -v

# JWT package
cd pkg/jwt
go test -v
```

### Specific Test

```bash
# Run single test
go test -v -run TestSendMessage_Success

# Run tests matching pattern
go test -v -run TestSendMessage
```

## Writing Tests

### Test Structure (AAA Pattern)

```go
func TestFunctionName_Scenario(t *testing.T) {
    // Arrange - Setup
    mockRepo := new(MockRepository)
    service := NewService(mockRepo)
    
    // Act - Execute
    result, err := service.DoSomething(input)
    
    // Assert - Verify
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Using Mocks

```go
// Create mock
mockRepo := new(MockRepository)

// Set expectations
mockRepo.On("GetByID", ctx, "id-123").Return(expectedData, nil)

// Use in test
service := NewService(mockRepo)
result, err := service.DoSomething(ctx, "id-123")

// Verify expectations were met
mockRepo.AssertExpectations(t)
```

### Table-Driven Tests

```go
func TestValidation(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected error
    }{
        {"empty input", "", ErrInvalidInput},
        {"valid input", "valid", nil},
        {"too long", strings.Repeat("a", 1000), ErrTooLong},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            err := Validate(tc.input)
            assert.Equal(t, tc.expected, err)
        })
    }
}
```

## Test Coverage

### View Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in terminal
go tool cover -func=coverage.out

# View in browser
go tool cover -html=coverage.out
```

### Coverage Goals

- Critical business logic: >85%
- Service layer: >80%
- Utilities: >90%

## Best Practices

### DO

✅ Test business logic, not implementation details
✅ Use descriptive test names: `TestFunctionName_Scenario`
✅ Test edge cases (empty strings, nil values, boundary conditions)
✅ Use table-driven tests for multiple scenarios
✅ Mock external dependencies (database, Kafka, Redis)
✅ Test error paths, not just happy paths
✅ Keep tests fast (< 1 second per test)

### DON'T

❌ Test private functions directly
❌ Test framework code (gin, gRPC)
❌ Test database queries (use integration tests)
❌ Use real external services in unit tests
❌ Share state between tests
❌ Ignore race conditions (`-race` flag)

## Testing Patterns

### Testing Service Layer

```go
func TestSendMessage_Success(t *testing.T) {
    // Mock dependencies
    mockRepo := new(MockMessageRepository)
    mockKafka := new(MockKafkaProducer)
    mockOutbox := new(MockOutboxRepository)
    
    service := NewMessageService(mockRepo, mockKafka, mockOutbox)
    
    // Setup expectations
    mockRepo.On("CreateWithChats", ctx, mock.AnythingOfType("*service.Message")).Return(nil)
    mockKafka.On("PublishMessageSent", ctx, mock.AnythingOfType("*service.Message")).Return(nil)
    
    // Execute
    msg, err := service.SendMessage(ctx, "", "user-1", "user-2", "content", "text")
    
    // Verify
    assert.NoError(t, err)
    assert.NotNil(t, msg)
    assert.Equal(t, "user-1", msg.SenderID)
    
    mockRepo.AssertExpectations(t)
    mockKafka.AssertExpectations(t)
}
```

### Testing Error Handling

```go
func TestSendMessage_InvalidInput(t *testing.T) {
    service := NewMessageService(mockRepo, mockKafka, mockOutbox)
    
    testCases := []struct {
        name        string
        senderID    string
        recipientID string
        content     string
    }{
        {"empty sender", "", "user-2", "content"},
        {"empty recipient", "user-1", "", "content"},
        {"empty content", "user-1", "user-2", ""},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            msg, err := service.SendMessage(ctx, "", tc.senderID, tc.recipientID, tc.content, "text")
            
            assert.Error(t, err)
            assert.Nil(t, msg)
        })
    }
}
```

### Testing Logging

```go
func TestSendMessage_LogsKafkaFailure(t *testing.T) {
    // Create test log handler
    testHandler := NewTestLogHandler()
    logger := slog.New(testHandler)
    slog.SetDefault(logger)
    defer slog.SetDefault(slog.Default())
    
    // Setup mocks to fail Kafka
    mockKafka.On("PublishMessageSent", ctx, mock.Anything).Return(errors.New("kafka error"))
    mockOutbox.On("SaveToOutbox", ctx, mock.Anything).Return(nil)
    
    // Execute
    service.SendMessage(ctx, "", "user-1", "user-2", "content", "text")
    
    // Verify log was written
    warnLog, found := testHandler.FindLog("Failed to publish to Kafka")
    assert.True(t, found)
    assert.Equal(t, "WARN", warnLog["level"])
}
```

## CI Integration

Tests run automatically in GitHub Actions:

```yaml
- name: Run tests
  run: |
    go test -v -race -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out
```

View results: https://github.com/666Stepan66612/ZeroMes/actions

## Troubleshooting

### Tests Fail with Race Condition

```bash
# Run with race detector
go test -race ./...

# Fix by adding proper synchronization
var mu sync.Mutex
mu.Lock()
defer mu.Unlock()
```

### Mock Not Called

```bash
# Check expectations
mockRepo.AssertExpectations(t)

# Common issues:
# - Wrong parameters in On()
# - Function not called
# - Called with different arguments
```

### Test Timeout

```bash
# Increase timeout
go test -timeout 30s ./...

# Or fix slow test
# - Remove sleep()
# - Mock external calls
# - Use smaller test data
```

## Resources

- [Testing in Go](https://go.dev/doc/tutorial/add-a-test)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)

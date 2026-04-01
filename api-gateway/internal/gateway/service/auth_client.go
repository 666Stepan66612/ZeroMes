package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	pkgjwt "github.com/666Stepan66612/ZeroMes/pkg/jwt"
)

type AuthClientService struct {
	secret string
	baseURL string
	httpClient *http.Client
}

func NewAuthClient(secret string, baseURL string) *AuthClientService {
    return &AuthClientService{
        secret:  secret,
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (c *AuthClientService) ValidateToken(token string) (string, error) {
	return pkgjwt.ValidateAccessToken(token, c.secret)
}

func (c *AuthClientService) ChangePassword(ctx context.Context, login, oldHash, newHash string) (string, error) {
	reqBody := map[string]string{
		"login":         login,
        "old_auth_hash": oldHash,
        "new_auth_hash": newHash,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/auth/change-password", bytes.NewBuffer(jsonData))
	if err != nil {
        return "", fmt.Errorf("failed to create request: %w", err)
    }

	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
        return "", fmt.Errorf("failed to execute request: %w", err)
    }
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
        return "", fmt.Errorf("failed to read response: %w", err)
    }

	if resp.StatusCode != http.StatusOK {
        var errResp struct {
            Error string `json:"error"`
        }
        json.Unmarshal(body, &errResp)
        return "", fmt.Errorf("auth service error: %s", errResp.Error)
    }

    var authResp struct {
        Success bool   `json:"success"`
        UserID  string `json:"user_id"`
    }
    if err := json.Unmarshal(body, &authResp); err != nil {
        return "", fmt.Errorf("failed to unmarshal response: %w", err)
    }

    if !authResp.Success {
        return "", fmt.Errorf("password change failed")
    }

	return authResp.UserID, nil
}
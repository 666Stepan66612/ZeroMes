package transport

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"auth-service/internal/auth/service"
	apperrors "auth-service/internal/cores/errors"
)

type Handler struct {
	authService service.AuthService
}

func NewHandler(authService service.AuthService) *Handler {
	return &Handler{
		authService: authService,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, apperrors.ErrInvalidPayload.Error())
		return
	}

	if len(strings.TrimSpace(req.Login)) < 3 || len(req.Login) > 32 {
		respondError(w, http.StatusBadRequest, "login must be 3–32 characters")
		return
	}
	if req.AuthHash == "" {
		respondError(w, http.StatusBadRequest, "auth_hash is required")
		return
	}
	if req.PublicKey == "" {
		respondError(w, http.StatusBadRequest, "public_key is required")
		return
	}

	user, tokens, err := h.authService.Register(
		r.Context(),
		req.Login,
		req.AuthHash,
		req.PublicKey,
	)
	if err != nil {
		if err == apperrors.ErrUserAlreadyExists {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		// Log the actual error for debugging
		println("Register error:", err.Error())
		respondError(w, http.StatusInternalServerError, apperrors.ErrInternalServer.Error())
		return
	}

	setTokenCookies(w, tokens.AccessToken, tokens.RefreshToken)
	respondJSON(w, http.StatusCreated, RegisterResponse{
		User: toUserDTO(user),
	})
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, err string) {
	respondJSON(w, status, ErrorResponse{Error: err})
}

func toUserDTO(user *service.UserPublic) UserDTO {
	return UserDTO{
		ID:        user.ID,
		Login:     user.Login,
		PublicKey: user.PublicKey,
		CreatedAt: user.CreatedAt,
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, apperrors.ErrInvalidPayload.Error())
		return
	}

	if strings.TrimSpace(req.Login) == "" {
		respondError(w, http.StatusBadRequest, "login is required")
		return
	}
	if req.AuthHash == "" {
		respondError(w, http.StatusBadRequest, "auth_hash is required")
		return
	}

	user, tokens, err := h.authService.Login(
		r.Context(),
		req.Login,
		req.AuthHash,
	)

	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	setTokenCookies(w, tokens.AccessToken, tokens.RefreshToken)
	respondJSON(w, http.StatusOK, LoginResponse{
		User: toUserDTO(user),
	})
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := r.Cookie("refresh_token")
	if err != nil {
		respondError(w, http.StatusUnauthorized, apperrors.ErrInvalidToken.Error())
		return
	}

	tokens, err := h.authService.RefreshToken(r.Context(), refreshToken.Value)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	setTokenCookies(w, tokens.AccessToken, tokens.RefreshToken)
	respondJSON(w, http.StatusOK, map[string]string{"message": "token refreshed"})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := r.Cookie("refresh_token")
	if err != nil {
		respondError(w, http.StatusUnauthorized, apperrors.ErrInvalidToken.Error())
		return
	}

	accessToken, err := r.Cookie("access_token")
	if err != nil {
		respondError(w, http.StatusUnauthorized, apperrors.ErrInvalidToken.Error())
		return
	}

	if err := h.authService.Logout(r.Context(), refreshToken.Value, accessToken.Value); err != nil {
		respondError(w, http.StatusInternalServerError, apperrors.ErrInternalServer.Error())
		return
	}

	clearTokenCookies(w)
	respondJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	login := r.URL.Query().Get("login")
	userID := r.URL.Query().Get("id")

	// Support search by either login or ID
	if login == "" && userID == "" {
		respondError(w, http.StatusBadRequest, "login or id parameter is required")
		return
	}

	if login != "" && len(login) < 3 {
		respondError(w, http.StatusBadRequest, "login must be at least 3 characters")
		return
	}

	var users []*service.UserPublic
	var err error

	if userID != "" {
		// Search by ID - get single user
		user, err := h.authService.GetByID(r.Context(), userID)
		if err != nil {
			respondJSON(w, http.StatusOK, SearchUserResponse{Users: []UserDTO{}})
			return
		}
		users = []*service.UserPublic{{
			ID:        user.ID,
			Login:     user.Login,
			PublicKey: user.PublicKey,
			CreatedAt: user.CreatedAt,
		}}
	} else {
		// Search by login
		users, err = h.authService.Search(r.Context(), login)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	dtos := make([]UserDTO, len(users))
	for i, u := range users {
		dtos[i] = toUserDTO(u)
	}

	respondJSON(w, http.StatusOK, SearchUserResponse{
		Users: dtos,
	})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, apperrors.ErrInvalidPayload.Error())
		return
	}

	userID, err := h.authService.ChangePassword(r.Context(), req.Login, req.OldAuthHash, req.NewAuthHash, req.NewPublicKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"user_id": userID,
	})
}

func setTokenCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   int(15 * time.Minute.Seconds()), // == access token
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		Path:     "/auth", // only for refresh endpoint
		MaxAge:   int(7 * 24 * time.Hour.Seconds()),
	})
}

func clearTokenCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		HttpOnly: true,
		Secure:   true,
		Path:     "/auth",
		MaxAge:   -1,
	})
}

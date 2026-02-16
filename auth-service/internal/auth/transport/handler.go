package transport

import (
	"encoding/json"
	"net/http"

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

	user, tokens, err := h.authService.Register(
		r.Context(),
		req.Login,
		req.AuthHash,
		req.PublicKey,
	)
	if err != nil {
		if err == apperrors.ErrUserAlreadyExists {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, apperrors.ErrInternalServer.Error())
		return
	}

	respondJSON(w, http.StatusCreated, RegisterResponse{
		User:         toUserDTO(user),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
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

	user, tokens, err := h.authService.Login(
		r.Context(),
		req.Login,
		req.AuthHash,
	)

	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, RegisterResponse{
		User:         toUserDTO(user),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, apperrors.ErrInvalidPayload.Error())
	}

	tokens, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, TokenPairResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, apperrors.ErrInvalidPayload.Error())
	}

	if err := h.authService.Logout(r.Context(), req.RefreshToken); err != nil {
		respondError(w, http.StatusInternalServerError, apperrors.ErrInternalServer.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

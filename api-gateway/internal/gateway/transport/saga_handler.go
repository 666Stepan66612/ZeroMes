package transport

import (
	"net/http"

	"api-gateway/internal/cores/domain"
	"api-gateway/internal/gateway/service"

	"github.com/gin-gonic/gin"
)

type SagaHandler struct {
	orchestrator service.Orchestrator
}

func NewSagaHandler(orchestrator service.Orchestrator) *SagaHandler {
	return &SagaHandler{
		orchestrator: orchestrator,
	}
}

func (h *SagaHandler) ChangePassword(c *gin.Context) {
	var req domain.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	if req.Login == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "login is required"})
		return
	}
	if req.OldAuthHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "old_auth_hash is required"})
		return
	}
	if req.NewAuthHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_auth_hash is required"})
		return
	}

	if req.NewPublicKey == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "new_public_key is required"})
    return
}

	resp, err := h.orchestrator.ChangePassword(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
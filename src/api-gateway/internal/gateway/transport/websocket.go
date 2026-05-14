package transport

import (
	"context"
	"net/http"
	"os"
	"strings"

	"api-gateway/internal/cores/domain"
	apperrors "api-gateway/internal/cores/errors"
	"api-gateway/internal/gateway/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	gatewayService service.GatewayService
}

func NewWebSocketHandler(gatewayService service.GatewayService) *WebSocketHandler {
	return &WebSocketHandler{
		gatewayService: gatewayService,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")

		// In production, check against whitelist
		allowedOrigins := getWhitelistedOrigins()
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}

		// Development mode fallback
		if os.Getenv("ENV") == "development" {
			return true
		}

		return false
	},
}

func getWhitelistedOrigins() []string {
	// Get from environment or config
	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	if originsEnv != "" {
		return strings.Split(originsEnv, ",")
	}

	// Default production origins
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "khmelev.site:8443"
	}

	return []string{
		"https://" + domain,
		"http://localhost:5173", // Local development
	}
}

func (h *WebSocketHandler) Handle(c *gin.Context) {
	println("[WebSocket] Handler called, userID:", c.GetString("userID"))
	println("[WebSocket] Attempting upgrade...")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		println("[WebSocket] Upgrade failed:", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": apperrors.ErrUpdate})
		return
	}
	defer conn.Close()

	println("[WebSocket] Upgrade successful!")

	userID := c.GetString("userID")
	token, _ := c.Cookie("access_token")
	ctx, cancel := context.WithCancel(context.WithValue(c.Request.Context(), domain.AccessTokenKey, token))
	defer cancel()

	send := make(chan []byte, 256)
	recv := make(chan []byte, 256)

	go func() {
		defer cancel()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			select {
			case recv <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case data := <-send:
				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					cancel()
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	h.gatewayService.HandleWebSocket(ctx, userID, send, recv)
}

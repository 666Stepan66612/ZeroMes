package transport

import (
	"context"
	"net/http"

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
		// Allow all origins for now (WebSocket is authenticated via JWT)
		// In production, you should whitelist specific origins
		return true
	},
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

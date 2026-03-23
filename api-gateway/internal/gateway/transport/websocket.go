package transport

import (
	"context"
	"net/http"

	"api-gateway/internal/cores/domain"
	"api-gateway/internal/gateway/service"
	apperrors "api-gateway/internal/cores/errors"

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
		return true
		/*
		TODO
		orgin := r.Header.Get("Origin")
		return origin == "https:frontend""
		(delete return true)
		*/
	},
}

func (h *WebSocketHandler) Handle(c *gin.Context) {
	conn, err := upgrader.Upgrade(c. Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": apperrors.ErrUpdate})
		return
	}
	defer conn.Close()

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
			case recv <-data:
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
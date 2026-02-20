package ws

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/terrascore/api/internal/agent"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/platform"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development; restrict in production via Kong
	},
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

// Handler handles WebSocket connections for real-time agent notifications.
type Handler struct {
	rdb       *redis.Client
	keycloak  *auth.KeycloakClient
	agentRepo *agent.Repository
	logger    *slog.Logger
}

// NewHandler creates a new WebSocket handler.
func NewHandler(rdb *redis.Client, keycloak *auth.KeycloakClient, agentRepo *agent.Repository, logger *slog.Logger) *Handler {
	return &Handler{
		rdb:       rdb,
		keycloak:  keycloak,
		agentRepo: agentRepo,
		logger:    logger,
	}
}

// ServeWS upgrades HTTP to WebSocket and streams agent offer notifications.
// Auth: JWT passed via ?token= query param (WebSocket can't use Authorization header from browser).
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "missing token query parameter")
		return
	}

	// Validate JWT
	claims, err := h.keycloak.ValidateToken(tokenStr)
	if err != nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "invalid token")
		return
	}

	// Extract keycloak_id from sub claim
	keycloakID, ok := (*claims)["sub"].(string)
	if !ok || keycloakID == "" {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "invalid token: missing sub")
		return
	}

	// Resolve agent
	ag, err := h.agentRepo.GetAgentByKeycloakID(r.Context(), keycloakID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "error", err)
		return
	}

	h.logger.Info("websocket connected", "agent_id", ag.ID)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Subscribe to Redis channels for this agent's real-time events
	agentID := ag.ID.String()
	sub := h.rdb.Subscribe(ctx,
		"agent:"+agentID+":offers",
		"agent:"+agentID+":events",
	)
	defer sub.Close()

	// Read pump: reads client messages for pong keepalive
	go h.readPump(ctx, cancel, conn)

	// Write pump: forwards Redis messages to WebSocket client
	h.writePump(ctx, conn, sub)

	h.logger.Info("websocket disconnected", "agent_id", ag.ID)
}

// readPump reads messages from the WebSocket client (for ping/pong keepalive).
func (h *Handler) readPump(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn) {
	defer cancel()

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				h.logger.Warn("websocket read error", "error", err)
			}
			return
		}
	}
}

// writePump forwards Redis pub/sub messages to the WebSocket client and sends pings.
func (h *Handler) writePump(ctx context.Context, conn *websocket.Conn, sub *redis.PubSub) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	ch := sub.Channel()

	for {
		select {
		case <-ctx.Done():
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return

		case msg, ok := <-ch:
			if !ok {
				conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}

			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				h.logger.Warn("websocket write error", "error", err)
				return
			}

		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/meglior/support-radar/server/internal/domain"
	"github.com/meglior/support-radar/server/internal/repository/postgres"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Handler struct {
	hub  *ConnectionHub
	repo *postgres.Repository
}

func NewHandler(hub *ConnectionHub, repo *postgres.Repository) *Handler {
	return &Handler{
		hub:  hub,
		repo: repo,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Printf("failed to read initial heartbeat: %v", err)
		return
	}

	var hb domain.Heartbeat
	if err := json.Unmarshal(msg, &hb); err != nil {
		log.Printf("malformed heartbeat: %v", err)
		return
	}

	if len(hb.IntegrityHash) != 64 {
		h.sendError(conn, domain.ErrInvalidSignature, "integrity_hash must be 64 chars")
		return
	}

	if hb.Status != domain.StatusOnline && hb.Status != domain.StatusMaintenance && hb.Status != domain.StatusFallback {
		h.sendError(conn, domain.ErrMalformedJSON, "invalid status value")
		return
	}

	ctx := r.Context()
	endpoint, err := h.repo.UpsertEndpoint(ctx, &hb)
	if err != nil {
		log.Printf("failed to upsert endpoint: %v", err)
		h.sendError(conn, domain.ErrMalformedJSON, "failed to register endpoint")
		return
	}

	ac := h.hub.Register(hb.MachineName, conn)
	ac.EndpointID = endpoint.ID
	h.hub.UpdateUUIDMapping(endpoint.ID, hb.MachineName)

	log.Printf("agent connected: %s (id=%s)", hb.MachineName, endpoint.ID)

	ack := map[string]string{"status": "registered", "endpoint_id": endpoint.ID}
	conn.WriteJSON(ack)

	for {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket error for %s: %v", hb.MachineName, err)
			}
			break
		}

		var heartbeat domain.Heartbeat
		if err := json.Unmarshal(msg, &heartbeat); err == nil {
			if _, err := h.repo.UpsertEndpoint(ctx, &heartbeat); err != nil {
				log.Printf("failed to update heartbeat: %v", err)
			}
			ac.LastPing = time.Now()
			continue
		}

		var response domain.CommandResponse
		if err := json.Unmarshal(msg, &response); err == nil {
			log.Printf("command response from %s: %s = %s", hb.MachineName, response.CommandID, response.Status)
			continue
		}

		var machineInfo domain.MachineInfo
		if err := json.Unmarshal(msg, &machineInfo); err == nil {
			log.Printf("machine info from %s: OS=%s", hb.MachineName, machineInfo.OSInfo.Caption)
			continue
		}

		log.Printf("unknown message from %s: %s", hb.MachineName, string(msg))
	}

	h.hub.Unregister(hb.MachineName)
	log.Printf("agent disconnected: %s", hb.MachineName)
}

func (h *Handler) sendError(conn *websocket.Conn, code, message string) {
	err := domain.ErrorResponse{
		ErrorCode:     code,
		Message:       message,
		Timestamp:     time.Now().Unix(),
		RetryAfterSec: 0,
	}
	conn.WriteJSON(err)
}

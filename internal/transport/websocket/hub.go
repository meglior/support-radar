package websocket

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/meglior/support-radar/server/internal/domain"
)

type AgentConnection struct {
	EndpointID  string
	MachineName string
	Conn        *websocket.Conn
	Mu          sync.Mutex
	LastPing    time.Time
}

type ConnectionHub struct {
	connections   sync.Map
	uuidToMachine sync.Map
}

func NewHub() *ConnectionHub {
	return &ConnectionHub{}
}

func (h *ConnectionHub) Register(machineName string, conn *websocket.Conn) *AgentConnection {
	ac := &AgentConnection{
		MachineName: machineName,
		Conn:        conn,
		LastPing:    time.Now(),
	}
	h.connections.Store(machineName, ac)
	return ac
}

func (h *ConnectionHub) Unregister(machineName string) {
	h.connections.Delete(machineName)
}

func (h *ConnectionHub) Get(machineName string) (*AgentConnection, bool) {
	val, ok := h.connections.Load(machineName)
	if !ok {
		return nil, false
	}
	return val.(*AgentConnection), true
}

func (h *ConnectionHub) SendCommand(machineName string, batch *domain.BatchRequest) error {
	ac, ok := h.Get(machineName)
	if !ok {
		return fmt.Errorf("agent not connected: %s", machineName)
	}

	ac.Mu.Lock()
	defer ac.Mu.Unlock()

	return ac.Conn.WriteJSON(batch)
}

func (h *ConnectionHub) SendCommandByEndpointID(endpointID string, batch *domain.BatchRequest) error {
	machineNameVal, ok := h.uuidToMachine.Load(endpointID)
	if !ok {
		return fmt.Errorf("endpoint not found: %s", endpointID)
	}
	return h.SendCommand(machineNameVal.(string), batch)
}

func (h *ConnectionHub) UpdateUUIDMapping(endpointID, machineName string) {
	h.uuidToMachine.Store(endpointID, machineName)
}

func (h *ConnectionHub) GetOnlineCount() int {
	count := 0
	h.connections.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

func (h *ConnectionHub) GetAllConnections() map[string]*AgentConnection {
	result := make(map[string]*AgentConnection)
	h.connections.Range(func(key, val interface{}) bool {
		result[key.(string)] = val.(*AgentConnection)
		return true
	})
	return result
}

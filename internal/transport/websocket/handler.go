package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/meglior/support-radar/server/internal/domain"
)

// EndpointRepository описывает интерфейс репозитория, необходимый для работы хендлера
type EndpointRepository interface {
	UpsertEndpoint(ctx context.Context, hb *domain.Heartbeat, ip string, domainName string) (*domain.Endpoint, error)
}

type Handler struct {
	repo     EndpointRepository
	upgrader websocket.Upgrader
}

// NewHandler создает новый экземпляр обработчика веб-сокетов
func NewHandler(repo EndpointRepository) *Handler {
	return &Handler{
		repo: repo,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Разрешаем любые Origin для тестов. В продакшене здесь должна быть валидация
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// HandleConnection обрабатывает входящие WebSocket-соединения от агентов
func (h *Handler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// 1. Апгрейдим HTTP соединение до протокола WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Ошибка апгрейда соединения: %v", err)
		return
	}
	defer conn.Close()

	// 2. Извлекаем реальный IP-адрес агента (с учетом проксирования вроде Nginx)
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = strings.Split(forwarded, ",")[0]
	} else {
		ip = strings.Split(ip, ":")[0]
	}

	log.Printf("[WS] Агент успешно подключился. IP: %s", ip)

	// 3. Цикл для непрерывного чтения хартбитов от Windows-агента
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[WS] Агент разорвал соединение или произошла ошибка чтения: %v", err)
			break
		}

		// Десериализуем входящий JSON в структуру Heartbeat
		var hb domain.Heartbeat
		if err := json.Unmarshal(message, &hb); err != nil {
			log.Printf("[WS] Ошибка парсинга JSON от агента: %v", err)
			continue
		}

		// Дефолтный домен (поскольку в легковесном Heartbeat поля домена нет)
		domainName := "WORKGROUP"

		// 4. Обновляем статус хоста в базе данных PostgreSQL
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		endpoint, err := h.repo.UpsertEndpoint(ctx, &hb, ip, domainName)
		cancel()

		if err != nil {
			log.Printf("[DB] Не удалось обновить данные хоста в базе: %v", err)
			continue
		}

		log.Printf("[SUCCESS] Хост '%s' (ID: %s) активен. Статус обновлен в БД.",
			endpoint.MachineName, endpoint.ID)
	}
}

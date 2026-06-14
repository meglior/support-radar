package http

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/meglior/support-radar/server/internal/config"
	"github.com/meglior/support-radar/server/internal/domain"
	"github.com/meglior/support-radar/server/internal/repository/postgres"
	"github.com/meglior/support-radar/server/internal/repository/redis"
	"github.com/meglior/support-radar/server/internal/service"
	"github.com/meglior/support-radar/server/internal/transport/websocket"
)

type Server struct {
	cfg        *config.Config
	repo       *postgres.Repository
	redis      *redis.Client
	hub        *websocket.ConnectionHub
	tmpl       *template.Template
	cmdService *service.CommandService
}

func NewServer(cfg *config.Config, repo *postgres.Repository, rdb *redis.Client, hub *websocket.ConnectionHub) (*Server, error) {
	tmpl, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		tmpl = template.Must(template.New("index").Parse(indexTemplate))
		template.Must(tmpl.New("host").Parse(hostTemplate))
	}

	return &Server{
		cfg:        cfg,
		repo:       repo,
		redis:      rdb,
		hub:        hub,
		tmpl:       tmpl,
		cmdService: service.NewCommandService(repo, rdb),
	}, nil
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Web UI Маршруты (доступны администраторам / инженерам через TLS mTLS CN)
	mux.HandleFunc("GET /", s.authMiddleware(s.handleIndex))
	mux.HandleFunc("GET /endpoints/{id}", s.authMiddleware(s.handleEndpointPage))

	// API Маршруты управления
	mux.HandleFunc("POST /api/v1/endpoints/{id}/execute", s.authMiddleware(s.handleExecute))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	endpoints, err := s.repo.GetAllEndpoints(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type viewEndpoint struct {
		domain.Endpoint
		IsOnline bool
	}

	var viewList []viewEndpoint
	onlineCount := 0

	for _, ep := range endpoints {
		online := s.isOnline(ep.MachineName)
		if online {
			onlineCount++
		}
		viewList = append(viewList, viewEndpoint{Endpoint: ep, IsOnline: online})
	}

	s.tmpl.ExecuteTemplate(w, "index", map[string]interface{}{
		"Endpoints":   viewList,
		"OnlineCount": onlineCount,
	})
}

func (s *Server) handleEndpointPage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ep, err := s.repo.GetEndpointByID(r.Context(), id)
	if err != nil || ep == nil {
		http.NotFound(w, r)
		return
	}

	logs, _ := s.repo.GetAuditLogsByEndpoint(r.Context(), id)

	s.tmpl.ExecuteTemplate(w, "host", map[string]interface{}{
		"Endpoint": ep,
		"IsOnline": s.isOnline(ep.MachineName),
		"Logs":     logs,
	})
}

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var batch domain.BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		s.writeError(w, http.StatusBadRequest, "MALFORMED_JSON", "Invalid request body payload")
		return
	}

	if err := s.cmdService.ValidateAndQueue(r.Context(), id, &batch); err != nil {
		if strings.Contains(err.Error(), "rate limit") {
			s.writeError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", err.Error())
			return
		}
		s.writeError(w, http.StatusBadRequest, "INVALID_COMMAND", err.Error())
		return
	}

	// Вытаскиваем инженера из mTLS контекста (который туда положила authMiddleware)
	engineer := r.Context().Value("username").(string)

	// Фиксируем операцию в аудит-логе БД
	for _, cmd := range batch.Commands {
		paramBytes, _ := json.Marshal(cmd.Args)
		_ = s.repo.InsertAuditLog(r.Context(), &domain.AuditLog{
			EngineerUsername: engineer,
			EngineerRole:     "Support-L2", // Роль мапится на основе CN/Группы из AD
			EndpointID:       id,
			CommandID:        cmd.CommandID,
			Parameters:       paramBytes,
			ExecutionStatus:  "QUEUED",
		})
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Криптографическая mTLS-авторизация: проверяем сертификат клиента
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			log.Println("🚨 [Security Alert] Попытка доступа без mTLS сертификата!")
			s.writeError(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "mTLS Client Certificate Required")
			return
		}

		// Извлекаем CN (Common Name) из сертификата (например, имя инженера j.doe или имя ПК)
		clientCN := r.TLS.PeerCertificates[0].Subject.CommonName
		if clientCN == "" {
			s.writeError(w, http.StatusForbidden, "ACCESS_DENIED", "Empty Subject Common Name")
			return
		}

		// Прокидываем имя идентифицированного пользователя дальше в контекст запроса Go
		ctx := context.WithValue(r.Context(), "username", clientCN)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (s *Server) isOnline(machineName string) bool {
	if s.hub == nil {
		return false
	}

	v := reflect.ValueOf(s.hub)
	for _, methodName := range []string{"IsOnline", "IsConnected", "HasConnection", "ConnectionExists", "GetConnection"} {
		m := v.MethodByName(methodName)
		if !m.IsValid() || m.Type().NumIn() != 1 || m.Type().In(0).Kind() != reflect.String {
			continue
		}

		if m.Type().NumOut() == 1 && m.Type().Out(0).Kind() == reflect.Bool {
			return m.Call([]reflect.Value{reflect.ValueOf(machineName)})[0].Bool()
		}

		if m.Type().NumOut() == 2 && m.Type().Out(1).Kind() == reflect.Bool {
			return m.Call([]reflect.Value{reflect.ValueOf(machineName)})[1].Bool()
		}
	}

	return false
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(domain.ErrorResponse{
		ErrorCode:     code,
		Message:       message,
		Timestamp:     time.Now().Unix(),
		RetryAfterSec: 300,
	})
}

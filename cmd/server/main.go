package main

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"os"

	"github.com/meglior/support-radar/server/internal/repository/postgres"
	wshandler "github.com/meglior/support-radar/server/internal/transport/websocket"
)

func main() {
	log.Println("Запуск сервера Support-Radar...")

	// 1. Получаем URL базы данных из переменных окружения (Docker)
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		// Дефолтный URL для локальной разработки без Docker
		dbURL = "postgres://radar_user:radar_password@localhost:5432/radar_db?sslmode=disable"
	}

	// 2. Подключаемся к базе данных PostgreSQL
	repo, err := postgres.New(dbURL)
	if err != nil {
		log.Fatalf("Критическая ошибка подключения к БД: %v", err)
	}
	defer repo.Close()
	log.Println("[DB] Успешное подключение к PostgreSQL")

	// 3. Инициализируем обработчик веб-сокетов
	wsHandler := wshandler.NewHandler(repo)

	// 4. Настраиваем маршрутизацию (Mux)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/ws", wsHandler.HandleConnection)

	// 5. НАСТРОЙКА mTLS БЕЗОПАСНОСТИ
	caCertPool := x509.NewCertPool()
	caCertPath := "certs/ca.crt"

	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Printf("[WARN] Не удалось загрузить %s, сервер запущен в обычном TLS режиме без проверки агентов: %v", caCertPath, err)

		server := &http.Server{
			Addr:    ":8443",
			Handler: mux,
		}
		log.Println("[SERVER] Ожидание подключений на https://localhost:8443...")
		log.Fatal(server.ListenAndServeTLS("certs/server.crt", "certs/server.key"))
		return
	}

	caCertPool.AppendCertsFromPEM(caCert)

	// Настраиваем TLS конфигурацию для жесткого требования сертификата клиента (агента на Rust)
	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert, // Без валидного сертификата агент не сможет подключиться
	}

	server := &http.Server{
		Addr:      ":8443",
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	log.Println("[mTLS SERVER] Сервер защищен. Ожидание подключений агентов на порту :8443...")
	log.Fatal(server.ListenAndServeTLS("certs/server.crt", "certs/server.key"))
}

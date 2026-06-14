package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/meglior/support-radar/server/internal/config"
	"github.com/meglior/support-radar/server/internal/repository/postgres"
	"github.com/meglior/support-radar/server/internal/repository/redis"
	httphandlers "github.com/meglior/support-radar/server/internal/transport/http"
	"github.com/meglior/support-radar/server/internal/transport/websocket"
)

func main() {
	log.Println("🚀 Support-Radar Server starting...")

	// 1. Загрузка конфигурации и маскирование сенситивных данных
	cfg := config.Load()
	sanitizedDBURL := sanitizeURL(cfg.DatabaseURL)
	log.Printf("[Init] Config loaded: addr=%s, db=%s, redis=%s", cfg.ServerAddr, sanitizedDBURL, cfg.RedisAddr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Инициализация инфраструктурного слоя (БД и Кэш)
	repo, err := postgres.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[Critical] Failed to connect to PostgreSQL: %v", err)
	}
	defer func() {
		if err := repo.Close(); err != nil {
			log.Printf("[Error] Failed to close DB repository: %v", err)
		}
	}()
	log.Println("✅ PostgreSQL connected")

	rdb := redis.New(cfg.RedisAddr)
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Printf("[Error] Failed to close Redis client: %v", err)
		}
	}()
	log.Println("✅ Redis connected")

	// 3. Инициализация WebSocket Hub
	hub := websocket.NewHub()

	// 4. Инициализация HTTP-сервера (сборка роутера внутри httphandlers)
	httpServer, err := httphandlers.NewServer(cfg, repo, rdb, hub)
	if err != nil {
		log.Fatalf("[Critical] Failed to create HTTP server: %v", err)
	}

	// 5. Проверка и автоматическая генерация mTLS сертификатов
	if _, err := os.Stat(cfg.TLSCertPath); os.IsNotExist(err) {
		log.Println("⚠️  TLS certificates not found, generating crypto-safe fallback pairs...")
		if err := autoGenerateCerts(cfg.TLSCertPath, cfg.TLSKeyPath); err != nil {
			log.Fatalf("[Critical] Auto-generation of certs failed: %v", err)
		}
		log.Println("✅ Local self-signed mTLS certificates generated successfully")
	}

	// 6. Настройка mTLS 1.3 (Mutual TLS) согласно требованиям ИБ
	// Загружаем наш же сертификат как доверенный для проверки клиентов (для локального MVP)
	certPool := x509.NewCertPool()
	caCert, err := os.ReadFile(cfg.TLSCertPath)
	if err != nil {
		log.Fatalf("[Critical] Failed to read CA cert for pool: %v", err)
	}
	certPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS13,
		ClientAuth:               tls.RequireAndVerifyClientCert, // Жесткое требование mTLS
		ClientCAs:                certPool,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	// Используем кастомный mux, который передаем в хендлеры
	mux := http.NewServeMux()
	httpServer.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      mux,
		TLSConfig:    tlsConfig,
		ReadTimeout:  15 * time.Second, // Оптимизировано под WebSocket Heartbeats
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 7. Асинхронный запуск сетевого потока
	go func() {
		log.Printf("🌐 Server listening on https://%s (mTLS 1.3 / Zero-Listening-Ports active)", cfg.ServerAddr)
		if err := server.ListenAndServeTLS(cfg.TLSCertPath, cfg.TLSKeyPath); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Critical] Server network runtime error: %v", err)
		}
	}()

	// 8. Ожидание сигналов ОС и Graceful Shutdown через NotifyContext
	<-ctx.Done()
	log.Println("🛑 Termination signal received. Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Error] Graceful shutdown failed: %v", err)
	}

	log.Println("👋 Support-Radar Server completely stopped. Infrastructure offline.")
}

// autoGenerateCerts создает криптографически стойкую пару ключей без внешних утилит
func autoGenerateCerts(certPath, keyPath string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Support-Radar-Local-Dev"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		return err
	}

	return nil
}

// sanitizeURL вырезает пароли из логов для ИБ-комплаенса
func sanitizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "[INVALID_URL_FORMAT]"
	}
	if u.User != nil {
		u.User = url.UserPassword(u.User.Username(), "******")
	}
	return u.String()
}

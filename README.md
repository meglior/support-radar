Часть 1:
# Support-Radar 📡

**Support-Radar** — это система автоматизированной диагностики и первой линии реагирования, предназначенная для обслуживания парка из **5000+ хостов** в корпоративной Windows-инфраструктуре. Проект нацелен на снижение нагрузки на 3-ю линию поддержки (L3) более чем на **50%** за счет автоматизации типовых исправлений и безопасного делегирования операций инженерам L1/L2.

## 🏗 Архитектура системы
Система строится по трехзвенной модели:
1.  **Central Server (Go):** Stateless-бэкенд, управляющий пулом WebSocket-соединений, ролевой моделью (RBAC) и аудитом.
2.  **Endpoint Agent (Rust Core):** Легковесная служба Windows, работающая с нулевым количеством прослушиваемых портов (Zero-Listening-Ports).
3.  **Web-UI:** Панель управления для инженеров с интеграцией Active Directory.

## 🛠 Технологический стек
*   **Backend:** Go (Core), PostgreSQL (Immutable Audit Log), Redis (Rate Limiting & Cache).
*   **Agent:** Rust (Tokio, WinAPI), PowerShell (динамические модули).
*   **Безопасность:** mTLS 1.3, Ed25519 (Emergency Fallback), AES-256/RSA (шифрование логов).

## 🚀 Ключевые возможности
*   **Hybrid Heartbeat:** Оптимизированный протокол обмена данными (~250–450 байт) для работы через медленные VPN-каналы.
*   **One-click Remediation:** Модуль автоматического исправления DNS, синхронизации времени и очистки дисков.
*   **Impersonation:** Безопасное выполнение пользовательских команд (Outlook, принтеры) через перехват токена активной сессии без использования SYSTEM-привилегий.
*   **Anti-tampering:** Постоянный контроль целостности бинарного файла агента с мгновенным алертингом в SIEM.

## 📁 Структура проекта (Go Server)
Согласно стандартам чистого кода, принятым в проекте:
*   `cmd/server` — Точка входа в приложение.
*   `internal/` — Бизнес-логика, handlers, репозитории и middleware.
*   `migrations/` — SQL-скрипты для PostgreSQL.
*   `scripts/` — Вспомогательные скрипты развертывания.

## 🛡 Безопасность
Все коммуникации защищены двусторонним TLS. Клиентские сертификаты выпускаются автоматически через **AD Auto-Enrollment**. Для критических ситуаций предусмотрен **Emergency Fallback** сервер, работающий по модели статического манифеста, подписанного Root-ключом разработчика.

Часть 2: Структура проекта Rust-агента
Hybrid Heartbeat и требованиями по безопасности (mTLS, Impersonation, Anti-tampering) структура проекта на Rust:

## support-radar-agent

```bash
support-radar-agent/
├── Cargo.toml
├── build.rs
├── src/
│   ├── main.rs
│   ├── app.rs
│   ├── domain/
│   │   ├── mod.rs
│   │   ├── models.rs
│   │   └── commands.rs
│   ├── infrastructure/
│   │   ├── mod.rs
│   │   ├── network/
│   │   │   ├── mtls.rs
│   │   │   └── websocket.rs
│   │   └── windows/
│   │       ├── impersonation.rs
│   │       ├── system_info.rs
│   │       └── process.rs
│   ├── security/
│   │   ├── mod.rs
│   │   ├── anti_tampering.rs
│   │   └── fallback.rs
│   └── modules/
│       ├── mod.rs
│       ├── remediation.rs
│       └── diagnostics.rs
└── tests/
```
Описание модулей
Корневые файлы

`Cargo.toml` — зависимости проекта (tokio, windows-sys, serde, rustls, sha2 и др.)
`build.rs` — скрипт сборки (компиляция манифеста Windows Service и ресурсов)

`src/`

`main.rs` — точка входа приложения, инициализация Service Control Manager (SCM)
`app.rs` — основной оркестратор: управление жизненным циклом и async-рантаймом

`domain/` — бизнес-логика и контракты

`models.rs` — основные модели (Heartbeat, MachineInfo)
`commands.rs` — статический маппинг Command ID

`infrastructure/` — низкоуровневая инфраструктура
`network/`

`mtls.rs` — работа с Windows Certificate Store и `mTLS 1.3`
`websocket.rs` — WebSocket-клиент с exponential backoff

`windows/`

`impersonation.rs` — имперсонация пользователей (WTSQueryUserToken, дублирование токенов)
`system_info.rs` — сбор системных метрик (CPU, диск C:, RAM)
`process.rs` — безопасный запуск процессов через CreateProcessAsUserW

`security/` — безопасность и отказоустойчивость

`anti_tampering.rs` — фоновый мониторинг целостности (SHA-256)
`fallback.rs` — Ed25519 верификация и аварийный канал обновлений (update.json)

`modules/` — функциональные модули

`remediation.rs` — выполнение команд remediation (CMD_FIX_DNS, CMD_SYNC_TIME и др.)
`diagnostics.rs` — сбор диагностической информации (SlowLogon, GPResult и т.д.)

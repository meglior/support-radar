Часть 1: README.md для репозитория Support-Radar
# Support-Radar 📡

**Support-Radar** — это система автоматизированной диагностики и первой линии реагирования, предназначенная для обслуживания парка из **5000+ хостов** в корпоративной Windows-инфраструктуре. Проект нацелен на снижение нагрузки на 3-ю линию поддержки (L3) более чем на **50%** за счет автоматизации типовых исправлений и безопасного делегирования операций инженерам L1/L2.

## 🏗 Архитектура системы
Система строится по трехзвенной модели:
1.  **Central Server (Go):** Stateless-бэкенд, управляющий пулом WebSocket-соединений, ролевой моделью (RBAC) и аудитом.
2.  **Endpoint Agent (Rust Core):** Легковесная служба Windows, работающая с нулевым количеством прослушиваемых портов (Zero-Listening-Ports) [5, 6].
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
Для обеспечения полной совместимости с моделью Hybrid Heartbeat и требованиями по безопасности (mTLS, Impersonation, Anti-tampering), рекомендуется следующая структура проекта на Rust
:
support-radar-agent/
├── Cargo.toml                # Зависимости: tokio, serde, windows-sys, rustls, sha2 [9, 16]
├── src/
│   ├── main.rs               # Инициализация службы Windows и асинхронного рантайма [9]
│   ├── client/
│   │   ├── mod.rs
│   │   ├── mtls.rs           # Настройка mTLS 1.3 и работа с Windows Certificate Store [19, 20]
│   │   └── websocket.rs      # Логика WebSocket, Ping/Pong и Exponential Backoff [25]
│   ├── protocol/
│   │   ├── mod.rs
│   │   ├── heartbeat.rs      # Структура Heartbeat (snake_case, Unix Timestamp) [13, 26]
│   │   └── machine_info.rs   # Детальная структура MachineInfo (Smart Update) [27, 28]
│   ├── commands/
│   │   ├── mod.rs            # Конструкция Match для ID команд [24, 29]
│   │   ├── executor.rs       # Запуск PowerShell из RAM без записи на диск [30]
│   │   └── impersonation.rs  # Логика WTSGetActiveConsoleSessionId и CreateProcessAsUserW [15]
│   ├── security/
│   │   ├── mod.rs
│   │   ├── integrity.rs      # Фоновый поток Anti-tampering (SHA-256 проверка) [16, 17]
│   │   └── fallback.rs       # Логика опроса Emergency Fallback через Ed25519 [12, 22]
│   └── utils/
│       └── logger.rs         # Запись событий в Windows Event Log (Application) [31, 32]
└── build.rs                  # Сборка ресурсов (иконка, манифест администратора)
Ключевые особенности этой структуры:
Модуль protocol/heartbeat.rs: Должен содержать struct Heartbeat с полями в snake_case, где timestamp имеет тип i64, а free_disk_space_gb — f64, как того требует спецификация сетевого протокола
.
Модуль security/integrity.rs: Реализует логику проверки хэша бинарника каждые 10 минут перед отправкой Smart Update
.
Модуль commands/impersonation.rs: Инкапсулирует небезопасные (unsafe) вызовы windows-sys для переключения контекста на активного пользователя, обеспечивая изоляцию привилегий при работе с пользовательскими данными
.
Сетевой слой client/websocket.rs: Использует асинхронные каналы (Channels) для обмена сообщениями, предотвращая блокировку основного потока службы при сетевых задержка

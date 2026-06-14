package domain

import (
	"time"
)

type Endpoint struct {
	ID            string    `json:"id"`
	MachineName   string    `json:"machine_name"`
	ActiveIP      string    `json:"active_ip"`
	DomainName    string    `json:"domain_name"`
	AgentVersion  string    `json:"agent_version"`
	IntegrityHash string    `json:"integrity_hash"`
	Status        string    `json:"status"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CreatedAt     time.Time `json:"created_at"`
}

type AuditLog struct {
	ID               int64     `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	EngineerUsername string    `json:"engineer_username"`
	EngineerRole     string    `json:"engineer_role"`
	EndpointID       string    `json:"endpoint_id"`
	CommandID        string    `json:"command_id"`
	Parameters       []byte    `json:"parameters"`
	ExecutionStatus  string    `json:"execution_status"`
}

type Heartbeat struct {
	AgentVersion    string  `json:"agent_version"`
	IntegrityHash   string  `json:"integrity_hash"`
	Timestamp       int64   `json:"timestamp"`
	MachineName     string  `json:"machine_name"`
	CurrentUser     string  `json:"current_user"`
	LastBoot        int64   `json:"last_boot"`
	FreeDiskSpaceGB float64 `json:"free_disk_space_gb"`
	Status          string  `json:"status"`
}

type MachineInfo struct {
	AgentVersion string   `json:"agent_version"`
	Timestamp    int64    `json:"timestamp"`
	MachineName  string   `json:"machine_name"`
	OSInfo       OSInfo   `json:"os_info"`
	Hardware     Hardware `json:"hardware"`
	Network      Network  `json:"network"`
	SessionInfo  Session  `json:"session_info"`
}

type OSInfo struct {
	Caption     string `json:"caption"`
	Version     string `json:"version"`
	BuildNumber string `json:"build_number"`
	InstallDate int64  `json:"install_date"`
}

type Hardware struct {
	CPUModel     string  `json:"cpu_model"`
	CoresLogical int     `json:"cores_logical"`
	RAMTotalGB   float64 `json:"ram_total_gb"`
	Disks        []Disk  `json:"disks"`
}

type Disk struct {
	Letter      string  `json:"letter"`
	TotalSizeGB float64 `json:"total_size_gb"`
	FreeSpaceGB float64 `json:"free_space_gb"`
	DriveType   string  `json:"drive_type"`
}

type Network struct {
	Domain     string `json:"domain"`
	ActiveIP   string `json:"active_ip"`
	MACAddress string `json:"mac_address"`
	VPNActive  bool   `json:"vpn_active"`
}

type Session struct {
	ActiveSessionID int    `json:"active_session_id"`
	CurrentUser     string `json:"current_user"`
	LogonTime       int64  `json:"logon_time"`
}

type CommandRequest struct {
	CommandID string                 `json:"command_id"`
	Args      map[string]interface{} `json:"args"`
}

type CommandResponse struct {
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
	Output    string `json:"output"`
	ExitCode  int    `json:"exit_code"`
}

type BatchRequest struct {
	TransactionID string           `json:"transaction_id"`
	Commands      []CommandRequest `json:"commands"`
}

type ErrorResponse struct {
	ErrorCode     string `json:"error_code"`
	Message       string `json:"message"`
	Timestamp     int64  `json:"timestamp"`
	RetryAfterSec int    `json:"retry_after_sec"`
}

const (
	StatusOnline      = "ONLINE"
	StatusMaintenance = "MAINTENANCE"
	StatusFallback    = "FALLBACK"
	StatusOffline     = "OFFLINE"
)

const (
	ExecutionPending = "PENDING"
	ExecutionSuccess = "SUCCESS"
	ExecutionFailed  = "FAILED"
)

const (
	ErrInvalidSignature    = "INVALID_SIGNATURE"
	ErrMalformedJSON       = "MALFORMED_JSON"
	ErrRateLimitExceeded   = "RATE_LIMIT_EXCEEDED"
	ErrCommandNotFound     = "COMMAND_NOT_FOUND"
	ErrImpersonationFailed = "IMPERSONATION_FAILED"
)

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/support-radar/server/internal/domain"
)

type Repository struct {
	db *sql.DB
}

func New(dbURL string) (*Repository, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) UpsertEndpoint(ctx context.Context, hb *domain.Heartbeat) (*domain.Endpoint, error) {
	query := `
		INSERT INTO endpoints (machine_name, active_ip, domain_name, agent_version, integrity_hash, status, last_heartbeat)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (machine_name) 
		DO UPDATE SET 
			active_ip = EXCLUDED.active_ip,
			domain_name = EXCLUDED.domain_name,
			agent_version = EXCLUDED.agent_version,
			integrity_hash = EXCLUDED.integrity_hash,
			status = EXCLUDED.status,
			last_heartbeat = NOW()
		RETURNING id, machine_name, active_ip, domain_name, agent_version, integrity_hash, status, last_heartbeat, created_at
	`

	var ep domain.Endpoint
	err := r.db.QueryRowContext(ctx, query,
		hb.MachineName,
		hb.Network.ActiveIP,
		hb.Network.Domain,
		hb.AgentVersion,
		hb.IntegrityHash,
		domain.StatusOnline,
	).Scan(
		&ep.ID, &ep.MachineName, &ep.ActiveIP, &ep.DomainName,
		&ep.AgentVersion, &ep.IntegrityHash, &ep.Status,
		&ep.LastHeartbeat, &ep.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("upsert endpoint failed: %w", err)
	}
	return &ep, nil
}

func (r *Repository) GetAllEndpoints(ctx context.Context) ([]domain.Endpoint, error) {
	query := `SELECT id, machine_name, active_ip, domain_name, agent_version, integrity_hash, status, last_heartbeat, created_at FROM endpoints ORDER BY machine_name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []domain.Endpoint
	for rows.Next() {
		var ep domain.Endpoint
		if err := rows.Scan(&ep.ID, &ep.MachineName, &ep.ActiveIP, &ep.DomainName, &ep.AgentVersion, &ep.IntegrityHash, &ep.Status, &ep.LastHeartbeat, &ep.CreatedAt); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints, rows.Err()
}

func (r *Repository) GetEndpointByID(ctx context.Context, id string) (*domain.Endpoint, error) {
	query := `SELECT id, machine_name, active_ip, domain_name, agent_version, integrity_hash, status, last_heartbeat, created_at FROM endpoints WHERE id = $1`

	var ep domain.Endpoint
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ep.ID, &ep.MachineName, &ep.ActiveIP, &ep.DomainName,
		&ep.AgentVersion, &ep.IntegrityHash, &ep.Status,
		&ep.LastHeartbeat, &ep.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ep, nil
}

func (r *Repository) InsertAuditLog(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (engineer_username, engineer_role, endpoint_id, command_id, parameters, execution_status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, timestamp
	`
	return r.db.QueryRowContext(ctx, query,
		log.EngineerUsername,
		log.EngineerRole,
		log.EndpointID,
		log.CommandID,
		log.Parameters,
		log.ExecutionStatus,
	).Scan(&log.ID, &log.Timestamp)
}

func (r *Repository) GetAuditLogsByEndpoint(ctx context.Context, endpointID string) ([]domain.AuditLog, error) {
	query := `
		SELECT id, timestamp, engineer_username, engineer_role, endpoint_id, command_id, parameters, execution_status
		FROM audit_logs WHERE endpoint_id = $1 ORDER BY timestamp DESC LIMIT 50`

	rows, err := r.db.QueryContext(ctx, query, endpointID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		if err := rows.Scan(&l.ID, &l.Timestamp, &l.EngineerUsername, &l.EngineerRole, &l.EndpointID, &l.CommandID, &l.Parameters, &l.ExecutionStatus); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

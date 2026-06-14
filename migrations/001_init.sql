CREATE TABLE IF NOT EXISTS endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    machine_name VARCHAR(128) UNIQUE NOT NULL,
    active_ip VARCHAR(45) NOT NULL,
    domain_name VARCHAR(255),
    agent_version VARCHAR(32) NOT NULL,
    integrity_hash CHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ONLINE',
    last_heartbeat TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_endpoints_search ON endpoints(machine_name, active_ip);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    engineer_username VARCHAR(128) NOT NULL,
    engineer_role VARCHAR(64) NOT NULL,
    endpoint_id UUID REFERENCES endpoints(id),
    command_id VARCHAR(64) NOT NULL,
    parameters JSONB,
    execution_status VARCHAR(32) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_endpoint ON audit_logs(endpoint_id, timestamp DESC);
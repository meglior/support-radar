package service

import (
	"context"
	"fmt"
	"time"

	"github.com/support-radar/server/internal/domain"
	"github.com/support-radar/server/internal/repository/postgres"
	"github.com/support-radar/server/internal/repository/redis"
)

type CommandService struct {
	repo  *postgres.Repository
	redis *redis.Client
}

func NewCommandService(repo *postgres.Repository, rdb *redis.Client) *CommandService {
	return &CommandService{repo: repo, redis: rdb}
}

func (s *CommandService) ValidateAndQueue(ctx context.Context, endpointID string, batch *domain.BatchRequest) error {
	endpoint, err := s.repo.GetEndpointByID(ctx, endpointID)
	if err != nil {
		return fmt.Errorf("failed to get endpoint: %w", err)
	}
	if endpoint == nil {
		return fmt.Errorf("endpoint not found")
	}

	allowed, _, err := s.redis.CheckRateLimit(ctx, endpoint.MachineName, 3, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("rate limit check failed: %w", err)
	}
	if !allowed {
		return fmt.Errorf("rate limit exceeded")
	}

	for _, cmd := range batch.Commands {
		if !IsValidCommandID(cmd.CommandID) {
			return fmt.Errorf("invalid command ID: %s", cmd.CommandID)
		}
	}
	return nil
}

func IsValidCommandID(cmdID string) bool {
	validCommands := map[string]bool{
		"CMD_FIX_DNS":         true,
		"CMD_SYNC_TIME":       true,
		"CMD_REPAIR_DOMAIN":   true,
		"CMD_CLEAN_TRASH":     true,
		"CMD_RESTART_SPOOLER": true,
		"CMD_GPUPDATE":        true,
		"CMD_RESET_PROFILE":   true,
		"CMD_REMAP_DRIVES":    true,
		"CMD_OUTLOOK_RESET":   true,
		"CMD_PRINTER_RESET":   true,
	}
	return validCommands[cmdID]
}

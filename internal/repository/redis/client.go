package redis

import (
	"context"
	"fmt"
	"time"

	redisclient "github.com/go-redis/redis/v8"
)

type Client struct {
	client *redisclient.Client
}

func New(addr string) *Client {
	return &Client{
		client: redisclient.NewClient(&redisclient.Options{
			Addr:     addr,
			Password: "",
			DB:       0,
		}),
	}
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) CheckRateLimit(ctx context.Context, machineName string, maxCommands int, window time.Duration) (bool, int, error) {
	key := fmt.Sprintf("rate_limit:%s", machineName)
	now := time.Now().UnixNano()
	windowStart := now - int64(window)

	// Используем Pipeline для атомарности выполнения операций скользящего окна
	pipe := c.client.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	cardResult := pipe.ZCard(ctx, key)
	pipe.ZRangeWithScores(ctx, key, 0, 0)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redisclient.Nil {
		return false, 0, err
	}

	count := cardResult.Val()
	if int(count) >= maxCommands {
		// Вычисляем оставшийся таймаут блокировки для ответа ErrorResponse.RetryAfterSec
		return false, int(window.Seconds()), nil
	}

	// Если лимит не превышен, добавляем текущую команду
	err = c.client.ZAdd(ctx, key, &redisclient.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d", now),
	}).Err()

	if err != nil {
		return false, 0, err
	}

	// Продлеваем TTL самого ключа в Redis, чтобы не захламлять память
	c.client.Expire(ctx, key, window)

	return true, 0, nil
}

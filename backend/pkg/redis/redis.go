package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"echo-union/backend/config"
)

// Client Redis 客户端封装
// 当前用于 Token 黑名单；后续可扩展缓存、分布式锁等场景
type Client struct {
	rdb    *goredis.Client
	logger *zap.Logger
}

// NewClient 创建 Redis 连接并执行 Ping 健康检查
func NewClient(cfg *config.RedisConfig, logger *zap.Logger) (*Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis 连接失败: %w", err)
	}

	logger.Info("Redis 连接成功", zap.String("addr", cfg.Addr))

	return &Client{rdb: rdb, logger: logger}, nil
}

// ── Token 黑名单 ──

const blacklistPrefix = "token:blacklist:"

// BlacklistToken 将 JWT ID 加入黑名单，TTL 与 Token 剩余有效期一致
func (c *Client) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil // Token 已过期，无需加入黑名单
	}
	return c.rdb.Set(ctx, blacklistPrefix+jti, "1", ttl).Err()
}

// IsBlacklisted 检查 JWT ID 是否在黑名单中
func (c *Client) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	n, err := c.rdb.Exists(ctx, blacklistPrefix+jti).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Close 关闭 Redis 连接
func (c *Client) Close() error {
	return c.rdb.Close()
}

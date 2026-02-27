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

// rateLimitScript 原子化限流 Lua 脚本：INCR + 首次设置 EXPIRE
// 避免 INCR 与 EXPIRE 分离执行的竞态条件（EXPIRE 失败导致 key 永不过期）
var rateLimitScript = goredis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
`)

// CheckRateLimit 基于 Redis Lua 脚本的原子速率限制
// 返回 true 表示允许请求，false 表示超限
func (c *Client) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	windowSec := int(window.Seconds())
	if windowSec <= 0 {
		windowSec = 1
	}
	result, err := rateLimitScript.Run(ctx, c.rdb, []string{key}, windowSec).Int64()
	if err != nil {
		return false, err
	}
	return result <= int64(limit), nil
}

// Ping 检查 Redis 连接是否正常
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Close 关闭 Redis 连接
func (c *Client) Close() error {
	return c.rdb.Close()
}

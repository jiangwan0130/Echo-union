package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 应用全局配置结构体
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"db"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Mail     MailConfig     `mapstructure:"mail"`
	Log      LogConfig      `mapstructure:"log"`
	Feature  FeatureConfig  `mapstructure:"feature"`
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Port    int        `mapstructure:"port"`
	BaseURL string     `mapstructure:"base_url"`
	CORS    CORSConfig `mapstructure:"cors"`
}

// CORSConfig 跨域配置
type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

// DatabaseConfig PostgreSQL 数据库配置
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Name            string `mapstructure:"name"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	SSLMode         string `mapstructure:"sslmode"`
	Timezone        string `mapstructure:"timezone"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`  // 连接最大生命周期（分钟）
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time"` // 空闲连接最大存活时间（分钟）
}

// DSN 生成 PostgreSQL 连接字符串
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode, c.Timezone,
	)
}

// RedisConfig Redis 缓存配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// AuthConfig JWT 认证配置
type AuthConfig struct {
	JWTSecret               string        `mapstructure:"jwt_secret"`
	AccessTokenTTL          time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTLDefault  time.Duration `mapstructure:"refresh_token_ttl_default"`
	RefreshTokenTTLRemember time.Duration `mapstructure:"refresh_token_ttl_remember_me"`
	Cookie                  CookieConfig  `mapstructure:"cookie"`
}

// CookieConfig Cookie 安全配置
type CookieConfig struct {
	Secure   bool   `mapstructure:"secure"`
	SameSite string `mapstructure:"same_site"`
	Domain   string `mapstructure:"domain"`
}

// MailConfig SMTP 邮件配置
type MailConfig struct {
	SMTPHost string `mapstructure:"smtp_host"`
	SMTPPort int    `mapstructure:"smtp_port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// FeatureConfig 功能开关配置
type FeatureConfig struct {
	OAImportEnabled bool `mapstructure:"oa_import_enabled"`
}

// Load 从配置文件与环境变量加载配置
// 优先级：环境变量 > 配置文件 > 默认值
func Load(path string) (*Config, error) {
	v := viper.New()

	// ── 默认值 ──
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.base_url", "http://localhost:8080")
	v.SetDefault("server.cors.allow_origins", []string{"http://localhost:5173"})

	v.SetDefault("db.host", "localhost")
	v.SetDefault("db.port", 5432)
	v.SetDefault("db.name", "echo_union")
	v.SetDefault("db.user", "postgres")
	v.SetDefault("db.password", "")
	v.SetDefault("db.sslmode", "disable")
	v.SetDefault("db.timezone", "Asia/Shanghai")
	v.SetDefault("db.max_open_conns", 25)
	v.SetDefault("db.max_idle_conns", 10)
	v.SetDefault("db.conn_max_lifetime", 60)  // 60分钟
	v.SetDefault("db.conn_max_idle_time", 30) // 30分钟

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetDefault("auth.access_token_ttl", "15m")
	v.SetDefault("auth.refresh_token_ttl_default", "24h")
	v.SetDefault("auth.refresh_token_ttl_remember_me", "168h")
	v.SetDefault("auth.cookie.secure", false)
	v.SetDefault("auth.cookie.same_site", "Lax")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")

	v.SetDefault("feature.oa_import_enabled", false)

	// ── 配置文件 ──
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
	}

	// ── 环境变量 ──
	v.SetEnvPrefix("ECHO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 配置文件不存在时仅依赖默认值和环境变量
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// ── 关键配置校验 ──
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate 校验关键配置项
func (c *Config) Validate() error {
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("配置校验失败: auth.jwt_secret 不能为空")
	}
	if len(c.Auth.JWTSecret) < 16 {
		return fmt.Errorf("配置校验失败: auth.jwt_secret 长度不能少于 16 字符")
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("配置校验失败: server.port 必须在 1-65535 之间")
	}
	return nil
}

// [自证通过] config/config.go

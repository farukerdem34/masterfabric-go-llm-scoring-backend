package config

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	JWT          JWTConfig
	Kafka        KafkaConfig
	WebSocket    WebSocketConfig
	Log          LogConfig
	RefreshToken RefreshTokenConfig
}

// WebSocketConfig holds real-time WebSocket settings.
type WebSocketConfig struct {
	Enabled         bool
	MaxConnections  int
	PingIntervalSec int
	ReadBufferSize  int
	WriteBufferSize int
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host              string
	Port              int
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	CORSAllowedOrigins []string
	MaxBodyBytes      int64
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DSN returns the PostgreSQL connection string with escaped credentials.
func (d DatabaseConfig) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   fmt.Sprintf("%s:%d", d.Host, d.Port),
		Path:   "/" + d.DBName,
	}
	u.RawQuery = url.Values{"sslmode": {d.SSLMode}}.Encode()
	return u.String()
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host            string
	Port            int
	Password        string
	DB              int
	PoolSize        int
	MinIdleConns    int
	ConnMaxIdleTime time.Duration
}

// Addr returns the Redis address string.
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// JWTConfig holds JWT signing settings.
type JWTConfig struct {
	Secret          string
	ExpirationHours int
	Issuer          string
}

// KafkaConfig holds Kafka connection and consumer settings.
type KafkaConfig struct {
	Brokers           []string
	GroupID           string
	Enabled           bool
	NumPartitions     int
	ReplicationFactor int
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// RefreshTokenConfig holds refresh token cookie and lifetime settings.
type RefreshTokenConfig struct {
	Duration       time.Duration // Token lifetime (default 7 days)
	CookieName     string        // Cookie name (default "refresh_token")
	CookiePath     string        // Cookie path (default "/auth/refresh")
	Secure         bool          // Secure flag (true in production)
	AccessTokenTTL int           // Access token lifetime in minutes (default 15)
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	cfg := &Config{
		Server: ServerConfig{
			Host:               envOrDefault("SERVER_HOST", "0.0.0.0"),
			Port:               envOrDefaultInt("PORT", envOrDefaultInt("SERVER_PORT", 8080)),
			ReadTimeout:        time.Duration(envOrDefaultInt("SERVER_READ_TIMEOUT_SECONDS", 15)) * time.Second,
			WriteTimeout:       time.Duration(envOrDefaultInt("SERVER_WRITE_TIMEOUT_SECONDS", 15)) * time.Second,
			IdleTimeout:        time.Duration(envOrDefaultInt("SERVER_IDLE_TIMEOUT_SECONDS", 60)) * time.Second,
			CORSAllowedOrigins: envOrDefaultSlice("CORS_ALLOWED_ORIGINS", nil),
			MaxBodyBytes:       envOrDefaultInt64("MAX_BODY_BYTES", 1<<20),
		},
		Database: loadDatabaseConfig(),
		Redis: RedisConfig{
			Host:            envOrDefault("REDIS_HOST", "localhost"),
			Port:            envOrDefaultInt("REDIS_PORT", 6379),
			Password:        envOrDefault("REDIS_PASSWORD", ""),
			DB:              envOrDefaultInt("REDIS_DB", 0),
			PoolSize:        envOrDefaultInt("REDIS_POOL_SIZE", 10*runtime.GOMAXPROCS(0)),
			MinIdleConns:    envOrDefaultInt("REDIS_MIN_IDLE_CONNS", 5),
			ConnMaxIdleTime: time.Duration(envOrDefaultInt("REDIS_CONN_MAX_IDLE_TIME_SECONDS", 300)) * time.Second,
		},
		JWT: JWTConfig{
			Secret:          envOrDefault("JWT_SECRET", "change-me-in-production"),
			ExpirationHours: envOrDefaultInt("JWT_EXPIRATION_HOURS", 24),
			Issuer:          envOrDefault("JWT_ISSUER", "masterfabric"),
		},
		Kafka: KafkaConfig{
			Brokers:           envOrDefaultSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			GroupID:           envOrDefault("KAFKA_GROUP_ID", "masterfabric-go"),
			Enabled:           envOrDefault("KAFKA_ENABLED", "false") == "true",
			NumPartitions:     envOrDefaultInt("KAFKA_NUM_PARTITIONS", 3),
			ReplicationFactor: envOrDefaultInt("KAFKA_REPLICATION_FACTOR", 1),
		},
		WebSocket: WebSocketConfig{
			Enabled:         envOrDefault("WS_ENABLED", "true") == "true",
			MaxConnections:  envOrDefaultInt("WS_MAX_CONNECTIONS", 1000),
			PingIntervalSec: envOrDefaultInt("WS_PING_INTERVAL_SECONDS", 30),
			ReadBufferSize:  envOrDefaultInt("WS_READ_BUFFER_SIZE", 1024),
			WriteBufferSize: envOrDefaultInt("WS_WRITE_BUFFER_SIZE", 1024),
		},
		Log: LogConfig{
			Level:  envOrDefault("LOG_LEVEL", "info"),
			Format: envOrDefault("LOG_FORMAT", "json"),
		},
		RefreshToken: RefreshTokenConfig{
			Duration:       7 * 24 * time.Hour,
			CookieName:     "refresh_token",
			CookiePath:     "/api/v1/auth/refresh",
			Secure:         envOrDefault("ENVIRONMENT", "development") == "production",
			AccessTokenTTL: 15,
		},
	}
	return cfg
}

func loadDatabaseConfig() DatabaseConfig {
	// Support Render's DATABASE_URL (single connection string)
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		return parseDatabaseURL(dbURL)
	}
	return DatabaseConfig{
		Host:            envOrDefault("DB_HOST", "localhost"),
		Port:            envOrDefaultInt("DB_PORT", 5432),
		User:            envOrDefault("DB_USER", "masterfabric"),
		Password:        envOrDefault("DB_PASSWORD", "masterfabric"),
		DBName:          envOrDefault("DB_NAME", "masterfabric"),
		SSLMode:         envOrDefault("DB_SSLMODE", "disable"),
		MaxConns:        envOrDefaultInt32("DB_MAX_CONNS", 25),
		MinConns:        envOrDefaultInt32("DB_MIN_CONNS", 5),
		ConnMaxLifetime: time.Duration(envOrDefaultInt("DB_CONN_MAX_LIFETIME_SECONDS", 3600)) * time.Second,
		ConnMaxIdleTime: time.Duration(envOrDefaultInt("DB_CONN_MAX_IDLE_TIME_SECONDS", 1800)) * time.Second,
	}
}

func parseDatabaseURL(dbURL string) DatabaseConfig {
	u, err := url.Parse(dbURL)
	if err != nil {
		// Fallback to individual env vars
		return DatabaseConfig{
			Host:     envOrDefault("DB_HOST", "localhost"),
			Port:     envOrDefaultInt("DB_PORT", 5432),
			User:     envOrDefault("DB_USER", "masterfabric"),
			Password: envOrDefault("DB_PASSWORD", "masterfabric"),
			DBName:   envOrDefault("DB_NAME", "masterfabric"),
			SSLMode:  envOrDefault("DB_SSLMODE", "disable"),
			MaxConns: envOrDefaultInt32("DB_MAX_CONNS", 25),
			MinConns: envOrDefaultInt32("DB_MIN_CONNS", 5),
		}
	}

	user := u.User.Username()
	password, _ := u.User.Password()
	host := u.Hostname()
	port := 5432
	if u.Port() != "" {
		if p, err := strconv.Atoi(u.Port()); err == nil {
			port = p
		}
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	sslmode := u.Query().Get("sslmode")
	if sslmode == "" {
		sslmode = "require"
	}

	return DatabaseConfig{
		Host:            host,
		Port:            port,
		User:            user,
		Password:        password,
		DBName:          dbName,
		SSLMode:         sslmode,
		MaxConns:        envOrDefaultInt32("DB_MAX_CONNS", 25),
		MinConns:        envOrDefaultInt32("DB_MIN_CONNS", 5),
		ConnMaxLifetime: time.Duration(envOrDefaultInt("DB_CONN_MAX_LIFETIME_SECONDS", 3600)) * time.Second,
		ConnMaxIdleTime: time.Duration(envOrDefaultInt("DB_CONN_MAX_IDLE_TIME_SECONDS", 1800)) * time.Second,
	}
}

func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func envOrDefaultInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func envOrDefaultInt32(key string, defaultVal int32) int32 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(n)
		}
	}
	return defaultVal
}

func envOrDefaultInt64(key string, defaultVal int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}

func envOrDefaultSlice(key string, defaultVal []string) []string {
	if val := os.Getenv(key); val != "" {
		parts := strings.Split(val, ",")
		var result []string
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultVal
}

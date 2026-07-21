package config

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "masterfabric", cfg.Database.User)
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
}

func TestLoad_EnvironmentOverrides(t *testing.T) {
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("DB_HOST", "db.example.com")
	defer os.Unsetenv("SERVER_PORT")
	defer os.Unsetenv("DB_HOST")

	cfg := Load()
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
}

func TestLoad_DBPoolInt32Bounds(t *testing.T) {
	os.Setenv("DB_MAX_CONNS", "50")
	os.Setenv("DB_MIN_CONNS", "2147483648")
	defer os.Unsetenv("DB_MAX_CONNS")
	defer os.Unsetenv("DB_MIN_CONNS")

	cfg := Load()
	assert.Equal(t, int32(50), cfg.Database.MaxConns)
	assert.Equal(t, int32(5), cfg.Database.MinConns)
}

func TestDatabaseConfig_DSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}
	expected := "postgres://user:pass@localhost:5432/testdb?sslmode=disable"
	assert.Equal(t, expected, cfg.DSN())
}

func TestDatabaseConfig_DSN_EscapesSpecialCharacters(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user@domain",
		Password: "p@ss:w?rd#",
		DBName:   "testdb",
		SSLMode:  "require",
	}
	dsn := cfg.DSN()
	assert.Contains(t, dsn, "postgres://")
	assert.Contains(t, dsn, "sslmode=require")
	assert.NotContains(t, dsn, "p@ss:w?rd#")
}

func TestRedisConfig_Addr(t *testing.T) {
	cfg := RedisConfig{Host: "redis.local", Port: 6380}
	assert.Equal(t, "redis.local:6380", cfg.Addr())
}

func TestLoad_DatabaseConnPoolDefaults(t *testing.T) {
	cfg := Load()
	assert.Equal(t, 1*time.Hour, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, 30*time.Minute, cfg.Database.ConnMaxIdleTime)
}

func TestLoad_DatabaseConnPoolOverrides(t *testing.T) {
	os.Setenv("DB_CONN_MAX_LIFETIME_SECONDS", "7200")
	os.Setenv("DB_CONN_MAX_IDLE_TIME_SECONDS", "600")
	defer os.Unsetenv("DB_CONN_MAX_LIFETIME_SECONDS")
	defer os.Unsetenv("DB_CONN_MAX_IDLE_TIME_SECONDS")

	cfg := Load()
	assert.Equal(t, 2*time.Hour, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, cfg.Database.ConnMaxIdleTime)
}

func TestLoad_RedisPoolDefaults(t *testing.T) {
	cfg := Load()
	assert.Equal(t, 10*runtime.GOMAXPROCS(0), cfg.Redis.PoolSize)
	assert.Equal(t, 5, cfg.Redis.MinIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.Redis.ConnMaxIdleTime)
}

func TestLoad_RedisPoolOverrides(t *testing.T) {
	os.Setenv("REDIS_POOL_SIZE", "20")
	os.Setenv("REDIS_MIN_IDLE_CONNS", "10")
	os.Setenv("REDIS_CONN_MAX_IDLE_TIME_SECONDS", "600")
	defer os.Unsetenv("REDIS_POOL_SIZE")
	defer os.Unsetenv("REDIS_MIN_IDLE_CONNS")
	defer os.Unsetenv("REDIS_CONN_MAX_IDLE_TIME_SECONDS")

	cfg := Load()
	assert.Equal(t, 20, cfg.Redis.PoolSize)
	assert.Equal(t, 10, cfg.Redis.MinIdleConns)
	assert.Equal(t, 10*time.Minute, cfg.Redis.ConnMaxIdleTime)
}

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaults(t *testing.T) {
	cfg := Default()
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "9090", cfg.Server.MetricsPort)
	assert.True(t, cfg.Server.Monitoring)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, 20, cfg.Pagination.DefaultLimit)
	assert.Equal(t, 100, cfg.Pagination.MaxLimit)
	assert.Equal(t, 100, cfg.Seeder.Count)
	assert.Equal(t, int32(10), cfg.Database.Pool.MaxConns)
}

func TestLoadOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  port: "9999"
  monitoring: false
database:
  url: postgres://u:p@h:5432/db
  pool:
    max_conns: 5
    max_conn_lifetime: 1h
log:
  level: debug
pagination:
  default_limit: 5
  max_limit: 50
seeder:
  count: 7
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "9999", cfg.Server.Port)
	assert.False(t, cfg.Server.Monitoring)
	assert.Equal(t, "postgres://u:p@h:5432/db", cfg.Database.URL)
	assert.Equal(t, int32(5), cfg.Database.Pool.MaxConns)
	assert.Equal(t, time.Hour, cfg.Database.Pool.MaxConnLifetime)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, 5, cfg.Pagination.DefaultLimit)
	assert.Equal(t, 50, cfg.Pagination.MaxLimit)
	assert.Equal(t, 7, cfg.Seeder.Count)

	// unspecified values keep their defaults
	assert.Equal(t, "9090", cfg.Server.MetricsPort)
	assert.Equal(t, int32(2), cfg.Database.Pool.MinConns)
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load("/does/not/exist.yaml")
	require.Error(t, err)
	assert.Equal(t, Default(), cfg)
}

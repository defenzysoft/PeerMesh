package peer

import (
	"testing"
	"time"
)

func TestNewConnectionPool(t *testing.T) {
	config := PoolConfig{
		MaxConnections: 50,
		Timeout:        30 * time.Second,
		KeepAlive:      60 * time.Second,
	}

	pool := NewConnectionPool(config)
	if pool == nil {
		t.Fatal("Connection pool is nil")
	}

	if len(pool.connections) != 0 {
		t.Error("Connection pool should be empty initially")
	}

	if pool.config.MaxConnections != config.MaxConnections {
		t.Errorf("Max connections mismatch: got %d, want %d", pool.config.MaxConnections, config.MaxConnections)
	}

	t.Logf("Connection pool created successfully")
}

func TestNewConnectionPoolDefaults(t *testing.T) {
	config := PoolConfig{}

	pool := NewConnectionPool(config)
	if pool == nil {
		t.Fatal("Connection pool is nil")
	}

	if pool.config.MaxConnections != 100 {
		t.Errorf("Default max connections mismatch: got %d, want 100", pool.config.MaxConnections)
	}

	if pool.config.Timeout != 30*time.Second {
		t.Errorf("Default timeout mismatch: got %v, want 30s", pool.config.Timeout)
	}

	t.Logf("Connection pool created with defaults successfully")
}

func TestConnectionPoolStats(t *testing.T) {
	config := PoolConfig{
		MaxConnections: 10,
		Timeout:        5 * time.Second,
		KeepAlive:      10 * time.Second,
	}

	pool := NewConnectionPool(config)
	if pool == nil {
		t.Fatalf("Failed to create connection pool")
	}

	stats := pool.GetStats()

	requiredStats := []string{"total_connections", "active_connections", "max_connections", "total_bytes_sent", "total_bytes_received", "timeout", "keep_alive"}
	for _, stat := range requiredStats {
		if _, exists := stats[stat]; !exists {
			t.Errorf("Missing stat: %s", stat)
		}
	}

	if stats["total_connections"] != 0 {
		t.Error("Total connections should be 0 initially")
	}

	if stats["active_connections"] != 0 {
		t.Error("Active connections should be 0 initially")
	}

	if stats["max_connections"] != config.MaxConnections {
		t.Errorf("Max connections mismatch: got %v, want %d", stats["max_connections"], config.MaxConnections)
	}

	t.Logf("Connection pool stats:")
	for key, value := range stats {
		t.Logf("  %s = %v", key, value)
	}
}

func TestConnectionPoolIsConnected(t *testing.T) {
	config := PoolConfig{
		MaxConnections: 5,
		Timeout:        1 * time.Second,
		KeepAlive:      2 * time.Second,
	}

	pool := NewConnectionPool(config)
	if pool == nil {
		t.Fatalf("Failed to create connection pool")
	}

	if pool.IsConnected("nonexistent") {
		t.Error("Non-existent peer should not be connected")
	}

	t.Logf("Connection pool IsConnected test passed")
}

func TestConnectionPoolGetConnection(t *testing.T) {
	config := PoolConfig{
		MaxConnections: 5,
		Timeout:        1 * time.Second,
		KeepAlive:      2 * time.Second,
	}

	pool := NewConnectionPool(config)
	if pool == nil {
		t.Fatalf("Failed to create connection pool")
	}

	conn := pool.GetConnection("nonexistent")
	if conn != nil {
		t.Error("Non-existent connection should return nil")
	}

	t.Logf("Connection pool GetConnection test passed")
}

func TestConnectionPoolGetAllConnections(t *testing.T) {
	config := PoolConfig{
		MaxConnections: 5,
		Timeout:        1 * time.Second,
		KeepAlive:      2 * time.Second,
	}

	pool := NewConnectionPool(config)
	if pool == nil {
		t.Fatalf("Failed to create connection pool")
	}

	connections := pool.GetAllConnections()
	if len(connections) != 0 {
		t.Errorf("Expected 0 connections, got %d", len(connections))
	}

	t.Logf("Connection pool GetAllConnections test passed")
}

func TestConnectionPoolDisconnectNonExistent(t *testing.T) {
	config := PoolConfig{
		MaxConnections: 5,
		Timeout:        1 * time.Second,
		KeepAlive:      2 * time.Second,
	}

	pool := NewConnectionPool(config)
	if pool == nil {
		t.Fatalf("Failed to create connection pool")
	}

	err := pool.Disconnect("nonexistent")
	if err == nil {
		t.Error("Expected error when disconnecting non-existent peer")
	}

	t.Logf("Connection pool Disconnect non-existent test passed")
}

func TestConnectionPoolClose(t *testing.T) {
	config := PoolConfig{
		MaxConnections: 5,
		Timeout:        1 * time.Second,
		KeepAlive:      2 * time.Second,
	}

	pool := NewConnectionPool(config)
	if pool == nil {
		t.Fatalf("Failed to create connection pool")
	}

	err := pool.Close()
	if err != nil {
		t.Errorf("Expected no error when closing empty pool: %v", err)
	}

	t.Logf("Connection pool Close test passed")
}

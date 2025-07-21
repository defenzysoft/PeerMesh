package discovery

import (
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	config := ServerConfig{
		Addr:              ":8080",
		HeartbeatInterval: 30 * time.Second,
		CleanupInterval:   60 * time.Second,
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("Server is nil")
	}

	if server.addr != config.Addr {
		t.Errorf("Address mismatch: got %s, want %s", server.addr, config.Addr)
	}

	if server.isRunning {
		t.Error("Server should not be running initially")
	}

	if len(server.peers) != 0 {
		t.Error("Server should have no peers initially")
	}

	t.Logf("Server created successfully:")
	t.Logf("  Address: %s", server.addr)
	t.Logf("  Running: %t", server.isRunning)
	t.Logf("  Peers: %d", len(server.peers))
}

func TestNewServerDefaults(t *testing.T) {
	config := ServerConfig{}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server with defaults: %v", err)
	}

	if server.addr != ":8080" {
		t.Errorf("Default address mismatch: got %s, want :8080", server.addr)
	}

	t.Logf("Server created with defaults successfully")
}

func TestServerStats(t *testing.T) {
	config := ServerConfig{
		Addr: ":8081",
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	stats := server.GetStats()

	requiredStats := []string{"running", "addr", "total_peers", "online_peers", "offline_peers"}
	for _, stat := range requiredStats {
		if _, exists := stats[stat]; !exists {
			t.Errorf("Missing stat: %s", stat)
		}
	}

	if stats["running"] != false {
		t.Error("Server should not be running initially")
	}

	if stats["addr"] != config.Addr {
		t.Errorf("Address mismatch: got %s, want %s", stats["addr"], config.Addr)
	}

	if stats["total_peers"] != 0 {
		t.Error("Total peers should be 0 initially")
	}

	t.Logf("Server stats:")
	for key, value := range stats {
		t.Logf("  %s = %v", key, value)
	}
}

func TestServerPeerManagement(t *testing.T) {
	config := ServerConfig{
		Addr: ":8082",
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	peers := server.GetPeers()
	if len(peers) != 0 {
		t.Errorf("Expected 0 peers, got %d", len(peers))
	}

	peer := server.GetPeer("nonexistent")
	if peer != nil {
		t.Error("Expected nil peer for nonexistent ID")
	}

	stats := server.GetStats()
	if stats["total_peers"] != 0 {
		t.Error("Total peers should be 0")
	}

	t.Logf("Server peer management test passed")
}

func TestServerIsRunning(t *testing.T) {
	config := ServerConfig{
		Addr: ":8083",
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server.IsRunning() {
		t.Error("Server should not be running initially")
	}

	t.Logf("Server IsRunning test passed")
}

func TestServerStartStop(t *testing.T) {
	config := ServerConfig{
		Addr: ":8084",
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	if !server.IsRunning() {
		t.Error("Server should be running after Start()")
	}

	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	if server.IsRunning() {
		t.Error("Server should not be running after Stop()")
	}

	t.Logf("Server Start/Stop test passed")
}

func TestServerDoubleStart(t *testing.T) {
	config := ServerConfig{
		Addr: ":8085",
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	err = server.Start()
	if err == nil {
		t.Error("Expected error when starting already running server")
	}

	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	t.Logf("Server double start test passed")
}

func TestServerDoubleStop(t *testing.T) {
	config := ServerConfig{
		Addr: ":8086",
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	err = server.Stop()
	if err != nil {
		t.Errorf("Expected no error when stopping non-running server: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	err = server.Stop()
	if err != nil {
		t.Errorf("Expected no error when stopping already stopped server: %v", err)
	}

	t.Logf("Server double stop test passed")
}

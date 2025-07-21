package peer

import (
	"testing"
	"time"
)

func TestNewNATTraversal(t *testing.T) {
	config := TraversalConfig{
		LocalPort:        0,
		STUNServers:      []string{"stun.l.google.com:19302"},
		HolePunchTimeout: 30 * time.Second,
	}

	traversal, err := NewNATTraversal(config)
	if err != nil {
		t.Fatalf("Failed to create NAT traversal: %v", err)
	}

	if traversal == nil {
		t.Fatal("NAT traversal is nil")
	}

	if len(traversal.peers) != 0 {
		t.Error("NAT traversal should have no peers initially")
	}

	if len(traversal.stunServers) == 0 {
		t.Error("STUN servers should not be empty")
	}

	t.Logf("NAT traversal created successfully")
}

func TestNewNATTraversalDefaults(t *testing.T) {
	config := TraversalConfig{}

	traversal, err := NewNATTraversal(config)
	if err != nil {
		t.Fatalf("Failed to create NAT traversal with defaults: %v", err)
	}

	if len(traversal.stunServers) == 0 {
		t.Error("Default STUN servers should not be empty")
	}

	t.Logf("NAT traversal created with defaults successfully")
}

func TestNATTraversalStats(t *testing.T) {
	config := TraversalConfig{
		LocalPort:        0,
		STUNServers:      []string{"stun.l.google.com:19302"},
		HolePunchTimeout: 30 * time.Second,
	}

	traversal, err := NewNATTraversal(config)
	if err != nil {
		t.Fatalf("Failed to create NAT traversal: %v", err)
	}

	stats := traversal.GetStats()

	requiredStats := []string{"local_address", "total_peers", "reachable_peers", "stun_servers"}
	for _, stat := range requiredStats {
		if _, exists := stats[stat]; !exists {
			t.Errorf("Missing stat: %s", stat)
		}
	}

	if stats["total_peers"] != 0 {
		t.Error("Total peers should be 0 initially")
	}

	if stats["reachable_peers"] != 0 {
		t.Error("Reachable peers should be 0 initially")
	}

	if len(stats["stun_servers"].([]string)) == 0 {
		t.Error("STUN servers should not be empty")
	}

	t.Logf("NAT traversal stats:")
	for key, value := range stats {
		t.Logf("  %s = %v", key, value)
	}
}

func TestNATTraversalIsPeerReachable(t *testing.T) {
	config := TraversalConfig{
		LocalPort:        0,
		STUNServers:      []string{"stun.l.google.com:19302"},
		HolePunchTimeout: 30 * time.Second,
	}

	traversal, err := NewNATTraversal(config)
	if err != nil {
		t.Fatalf("Failed to create NAT traversal: %v", err)
	}

	if traversal.IsPeerReachable("nonexistent") {
		t.Error("Non-existent peer should not be reachable")
	}

	t.Logf("NAT traversal IsPeerReachable test passed")
}

func TestNATTraversalGetPeerInfo(t *testing.T) {
	config := TraversalConfig{
		LocalPort:        0,
		STUNServers:      []string{"stun.l.google.com:19302"},
		HolePunchTimeout: 30 * time.Second,
	}

	traversal, err := NewNATTraversal(config)
	if err != nil {
		t.Fatalf("Failed to create NAT traversal: %v", err)
	}

	peerInfo := traversal.GetPeerInfo("nonexistent")
	if peerInfo != nil {
		t.Error("Non-existent peer should return nil")
	}

	t.Logf("NAT traversal GetPeerInfo test passed")
}

func TestNATTraversalGetAllPeers(t *testing.T) {
	config := TraversalConfig{
		LocalPort:        0,
		STUNServers:      []string{"stun.l.google.com:19302"},
		HolePunchTimeout: 30 * time.Second,
	}

	traversal, err := NewNATTraversal(config)
	if err != nil {
		t.Fatalf("Failed to create NAT traversal: %v", err)
	}

	peers := traversal.GetAllPeers()
	if len(peers) != 0 {
		t.Errorf("Expected 0 peers, got %d", len(peers))
	}

	t.Logf("NAT traversal GetAllPeers test passed")
}

func TestNATTraversalClose(t *testing.T) {
	config := TraversalConfig{
		LocalPort:        0,
		STUNServers:      []string{"stun.l.google.com:19302"},
		HolePunchTimeout: 30 * time.Second,
	}

	traversal, err := NewNATTraversal(config)
	if err != nil {
		t.Fatalf("Failed to create NAT traversal: %v", err)
	}

	err = traversal.Close()
	if err != nil {
		t.Errorf("Expected no error when closing traversal: %v", err)
	}

	t.Logf("NAT traversal Close test passed")
}

func TestCreateSTUNRequest(t *testing.T) {
	config := TraversalConfig{
		LocalPort:        0,
		STUNServers:      []string{"stun.l.google.com:19302"},
		HolePunchTimeout: 30 * time.Second,
	}

	traversal, err := NewNATTraversal(config)
	if err != nil {
		t.Fatalf("Failed to create NAT traversal: %v", err)
	}

	request := traversal.createSTUNRequest()
	if len(request) != 20 {
		t.Errorf("STUN request should be 20 bytes, got %d", len(request))
	}

	messageType := (uint16(request[0]) << 8) | uint16(request[1])
	if messageType != 0x0001 {
		t.Errorf("Expected message type 0x0001, got 0x%04x", messageType)
	}

	t.Logf("STUN request creation test passed")
}

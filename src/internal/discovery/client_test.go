package discovery

import (
	"net"
	"testing"
)

func TestNewClient(t *testing.T) {
	config := ClientConfig{
		ServerURL: "ws://localhost:8080",
		PeerID:    "test-peer-1",
		LocalIP:   net.ParseIP("192.168.1.100"),
		Reconnect: true,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client is nil")
	}

	if client.serverURL != config.ServerURL {
		t.Errorf("Server URL mismatch: got %s, want %s", client.serverURL, config.ServerURL)
	}

	if client.peerID != config.PeerID {
		t.Errorf("Peer ID mismatch: got %s, want %s", client.peerID, config.PeerID)
	}

	if !client.localIP.Equal(config.LocalIP) {
		t.Errorf("Local IP mismatch: got %s, want %s", client.localIP.String(), config.LocalIP.String())
	}

	if client.reconnect != config.Reconnect {
		t.Errorf("Reconnect mismatch: got %v, want %v", client.reconnect, config.Reconnect)
	}

	if client.IsConnected() {
		t.Error("Client should not be connected initially")
	}

	t.Logf("Client created successfully:")
	t.Logf("  ServerURL: %s", client.serverURL)
	t.Logf("  Peer ID: %s", client.peerID)
	t.Logf("  Local IP: %s", client.localIP.String())
	t.Logf("  Reconnect: %t", client.reconnect)
}

func TestNewClientValidation(t *testing.T) {
	config := ClientConfig{
		ServerURL: "",
		PeerID:    "test-peer",
		LocalIP:   net.ParseIP("192.168.1.100"),
	}

	_, err := NewClient(config)
	if err == nil {
		t.Error("NewClient error for empty server URL")
	}

	config = ClientConfig{
		ServerURL: "ws://localhost:8080",
		PeerID:    "",
		LocalIP:   net.ParseIP("192.168.1.100"),
	}

	_, err = NewClient(config)
	if err == nil {
		t.Error("Expected error for empty peer ID")
	}
}

func TestClientStats(t *testing.T) {
	config := ClientConfig{
		ServerURL: "ws://localhost:8080",
		PeerID:    "test-peer-2",
		LocalIP:   net.ParseIP("192.168.1.101"),
		Reconnect: false,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	stats := client.GetStats()

	requiredStats := []string{"connected", "peer_id", "server_url", "total_peers", "online_peers", "offline_peers"}
	for _, stat := range requiredStats {
		if _, exists := stats[stat]; !exists {
			t.Errorf("Missing stat: %s", stat)
		}
	}

	if stats["connected"] != false {
		t.Error("Client should not be connected initially")
	}

	if stats["peer_id"] != config.PeerID {
		t.Errorf("Peer ID mismatch: got %s, want %s", stats["peer_id"], config.PeerID)
	}

	if stats["server_url"] != config.ServerURL {
		t.Errorf("Server URL mismatch: got %s, want %s", stats["server_url"], config.ServerURL)
	}

	if stats["total_peers"] != 0 {
		t.Error("Total peers should be 0 initially")
	}

	t.Logf("Client stats:")
	for key, value := range stats {
		t.Logf("  %s = %v", key, value)
	}
}

func TestPeerManagement(t *testing.T) {
	config := ClientConfig{
		ServerURL: "ws://localhost:8080",
		PeerID:    "test-peer-3",
		LocalIP:   net.ParseIP("192.168.1.102"),
		Reconnect: false,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	peers := client.GetPeers()
	if len(peers) != 0 {
		t.Errorf("Excepted 0 peers, got %d", len(peers))
	}

	peer := client.GetPeer("nonexistent")
	if peer != nil {
		t.Error("Expected nil peer for nonexistent ID")
	}

	stats := client.GetStats()
	if stats["total_peers"] != 0 {
		t.Error("Total peers should be 0")
	}

	t.Logf("Peer management test passed")
}

func TestMessageHandlerRegistration(t *testing.T) {
	config := ClientConfig{
		ServerURL: "ws://localhost:8080",
		PeerID:    "test-peer-4",
		LocalIP:   net.ParseIP("192.168.1.103"),
		Reconnect: false,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	customHandlerCalled := false
	customHandler := func(msg *Message) error {
		customHandlerCalled = true
		return nil
	}

	client.RegisterMessageHandler("custom_type", customHandler)

	testMsg := &Message{
		Type: "custom_type",
		Payload: map[string]interface{}{
			"test": "data",
		},
	}

	err = client.handleMessage(testMsg)
	if err != nil {
		t.Fatalf("Failed to handle message: %v", err)
	}

	if !customHandlerCalled {
		t.Error("Custom handler was not called")
	}

	t.Logf("Custom handler registration test passed")
}

func TestMessageValidation(t *testing.T) {
	config := ClientConfig{
		ServerURL: "ws://localhost:8080",
		PeerID:    "test-peer-5",
		LocalIP:   net.ParseIP("192.168.1.104"),
		Reconnect: false,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	unknownMessage := &Message{
		Type: "unknown_type",
		Payload: map[string]interface{}{
			"test": "data",
		},
	}

	err = client.handleMessage(unknownMessage)
	if err == nil {
		t.Error("Expected error for unknown message type")
	}

	t.Logf("Message validation test passed")
}

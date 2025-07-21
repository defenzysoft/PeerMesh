package vpn

import (
	_ "net"
	"testing"
)

func TestNewManager(t *testing.T) {
	config := ManagerConfig{
		SubnetCIDR: "10.0.0.0/24",
		MTU:        1500,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Manager is nil")
	}

	if manager.localIP == nil {
		t.Error("Local IP is nil")
	}

	if manager.gatewayIP == nil {
		t.Error("Gateway IP is nil")
	}

	if manager.subnet == nil {
		t.Error("Subnet is nil")
	}

	if !manager.subnet.Contains(manager.localIP) {
		t.Errorf("Local IP %s is not in subnet %s", manager.localIP.String(), manager.subnet.String())
	}

	if !manager.subnet.Contains(manager.gatewayIP) {
		t.Errorf("Gateway IP %s is not in subnet %s", manager.gatewayIP.String(), manager.subnet.String())
	}

	t.Logf("Manager created successfully:")
	t.Logf("  Local IP: %s", manager.localIP.String())
	t.Logf("  Gateway IP: %s", manager.gatewayIP.String())
	t.Logf("  Subnet: %s", manager.subnet.String())
}

func TestIPAllocation(t *testing.T) {
	config := ManagerConfig{
		SubnetCIDR: "192.168.1.0/24",
		MTU:        1500,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	peerIP, err := manager.AllocatePeerIP()
	if err != nil {
		t.Fatalf("Failed to allocate peer IP: %v", err)
	}

	if peerIP == nil {
		t.Fatal("Allocated IP is nil")
	}

	if !manager.subnet.Contains(peerIP) {
		t.Errorf("Allocated IP %s is not in subnet %s", peerIP.String(), manager.subnet.String())
	}

	if peerIP.Equal(manager.localIP) {
		t.Error("Allocated IP is the same as local IP")
	}

	if peerIP.Equal(manager.gatewayIP) {
		t.Error("Allocated IP is the same as gateway IP")
	}

	err = manager.ReleasePeerIP(peerIP)
	if err != nil {
		t.Fatalf("Failed to release peer IP: %v", err)
	}

	t.Logf("IP allocation test passed:")
	t.Logf("  Allocated IP: %s", peerIP.String())
}

func TestManagerStats(t *testing.T) {
	config := ManagerConfig{
		SubnetCIDR: "172.16.0.0/24",
		MTU:        1500,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	stats := manager.GetStats()

	requiredStats := []string{"running", "local_ip", "gateway_ip", "subnet", "available_ips", "allocated_ips"}
	for _, stat := range requiredStats {
		if _, exists := stats[stat]; !exists {
			t.Errorf("Missing stat: %s", stat)
		}
	}

	if stats["running"] != false {
		t.Error("Manager should not be running initially")
	}

	if stats["local_ip"] == "" {
		t.Error("Local IP should not be empty")
	}

	if stats["available_ips"].(int) <= 0 {
		t.Error("Available IPs should be positive")
	}

	t.Logf("Manager stats:")
	for key, value := range stats {
		t.Logf("  %s: %v", key, value)
	}
}

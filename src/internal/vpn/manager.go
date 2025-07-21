package vpn

import (
	"fmt"
	"log"
	"net"
	"sync"
)

// Manager represents the main VPN manager that coordinates all components
type Manager struct {
	interface_     *Interface
	allocator      *IPAllocator
	processor      *PacketProcessor
	subnet         *net.IPNet
	localIP        net.IP
	gatewayIP      net.IP
	isRunning      bool
	mutex          sync.RWMutex
	stopChan       chan struct{}
	packetHandlers map[PacketType]PacketHandler
}

// ManagerConfig holds configuration for the VPN manager
type ManagerConfig struct {
	SubnetCIDR string
	LocalIP    string
	MTU        int
}

// NewManager creates a new VPN manager
func NewManager(config ManagerConfig) (*Manager, error) {
	_, subnet, err := net.ParseCIDR(config.SubnetCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet CIDR: %w", err)
	}

	allocator, err := NewIPAllocator(subnet)
	if err != nil {
		return nil, fmt.Errorf("failed to create IP allocator: %w", err)
	}

	var localIP net.IP
	if config.LocalIP != "" {
		ip := net.ParseIP(config.LocalIP)
		if ip == nil {
			return nil, fmt.Errorf("invalid local IP: %s", config.LocalIP)
		}
		err = allocator.AllocateSpecificIP(ip)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate local IP: %w", err)
		}
		localIP = ip
	} else {
		localIP, err = allocator.AllocateIP()
		if err != nil {
			return nil, fmt.Errorf("failed to allocate local IP: %w", err)
		}
	}

	gatewayIP := allocator.GetGatewayIP()

	if gatewayIP.Equal(localIP) {
		gatewayIP = allocator.incrementIP(localIP)
	}

	processor := NewPacketProcessor(2048)

	interfaceConfig := InterfaceConfig{
		Name:    "peermesh0",
		IP:      localIP,
		Netmask: subnet.Mask,
		Gateway: gatewayIP,
		MTU:     config.MTU,
	}

	interface_, err := NewInterface(interfaceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create interface: %w", err)
	}

	manager := &Manager{
		interface_:     interface_,
		allocator:      allocator,
		processor:      processor,
		subnet:         subnet,
		localIP:        localIP,
		gatewayIP:      gatewayIP,
		stopChan:       make(chan struct{}),
		packetHandlers: make(map[PacketType]PacketHandler),
	}

	manager.registerDefaultHandlers()

	return manager, nil
}

// Start starts the VPN manager
func (m *Manager) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.isRunning {
		return fmt.Errorf("VPN manager is already running")
	}

	err := m.interface_.Open()
	if err != nil {
		return fmt.Errorf("failed to open interface: %w", err)
	}

	m.isRunning = true
	go m.run()

	log.Printf("VPN manager started with local IP: %s", m.localIP.String())
	return nil
}

// Stop stops the VPN manager
func (m *Manager) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.isRunning {
		return nil
	}

	m.isRunning = false
	close(m.stopChan)

	err := m.interface_.Close()
	if err != nil {
		return fmt.Errorf("failed to close interface: %w", err)
	}

	log.Println("VPN manager stopped")
	return nil
}

// IsRunning returns whether the VPN manager is running
func (m *Manager) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.isRunning
}

// GetLocalIP returns the local IP address
func (m *Manager) GetLocalIP() net.IP {
	return m.localIP
}

// GetGatewayIP returns the gateway IP address
func (m *Manager) GetGatewayIP() net.IP {
	return m.gatewayIP
}

// GetSubnet returns the subnet
func (m *Manager) GetSubnet() *net.IPNet {
	return m.subnet
}

// AllocatePeerIP allocates an IP address for a peer
func (m *Manager) AllocatePeerIP() (net.IP, error) {
	return m.allocator.AllocateIP()
}

// ReleasePeerIP releases an IP address allocated to a peer
func (m *Manager) ReleasePeerIP(ip net.IP) error {
	return m.allocator.ReleaseIP(ip)
}

// RegisterPacketHandler registers a custom packet handler
func (m *Manager) RegisterPacketHandler(packetType PacketType, handler PacketHandler) {
	m.packetHandlers[packetType] = handler
	m.processor.RegisterHandler(packetType, handler)
}

// run is the main loop for the VPN manager
func (m *Manager) run() {
	for {
		select {
		case <-m.stopChan:
			return
		default:
			packet, err := m.interface_.Read()
			if err != nil {
				continue
			}

			err = m.processor.ProcessPacket(packet)
			if err != nil {
				log.Printf("Failed to process packet: %v", err)
			}
		}
	}
}

// registerDefaultHandlers registers default packet handlers
func (m *Manager) registerDefaultHandlers() {
	m.RegisterPacketHandler(PacketTypeICMP, &ICMPHandler{manager: m})
	m.RegisterPacketHandler(PacketTypeTCP, &TCPHandler{manager: m})
	m.RegisterPacketHandler(PacketTypeUDP, &UDPHandler{manager: m})
}

// WritePacket writes a packet to the interface
func (m *Manager) WritePacket(packet []byte) error {
	if !m.IsRunning() {
		return fmt.Errorf("VPN manager is not running")
	}
	return m.interface_.Write(packet)
}

// GetStats returns basic statistics
func (m *Manager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"running":       m.IsRunning(),
		"local_ip":      m.localIP.String(),
		"gateway_ip":    m.gatewayIP.String(),
		"subnet":        m.subnet.String(),
		"available_ips": m.allocator.GetAvailableCount(),
		"allocated_ips": len(m.allocator.GetAllocatedIPs()),
	}
}

package vpn

import (
	"fmt"
	"net"
	"os"
	"runtime"
)

type Interface struct {
	Name      string
	IP        net.IP
	Netmask   net.IPMask
	Gateway   net.IP
	MTU       int
	file      *os.File
	isOpen    bool
	readChan  chan []byte
	writeChan chan []byte
}

type InterfaceConfig struct {
	Name    string
	IP      net.IP
	Netmask net.IPMask
	Gateway net.IP
	MTU     int
}

// NewInterface creates a new TUN interface with the given configuration
func NewInterface(config InterfaceConfig) (*Interface, error) {
	if config.MTU <= 0 {
		config.MTU = 1500
	}

	if config.Name == "" {
		config.Name = "peermesh0"
	}

	if config.IP == nil {
		return nil, fmt.Errorf("IP address is required")
	}

	if config.Netmask == nil {
		config.Netmask = net.CIDRMask(24, 32)
	}

	i := &Interface{
		Name:      config.Name,
		IP:        config.IP,
		Netmask:   config.Netmask,
		Gateway:   config.Gateway,
		MTU:       config.MTU,
		readChan:  make(chan []byte, 100),
		writeChan: make(chan []byte, 100),
	}

	return i, nil
}

// Open creates and opens the TUN interface
func (i *Interface) Open() error {
	if i.isOpen {
		return fmt.Errorf("interface already open")
	}

	var err error
	switch runtime.GOOS {
	case "linux":
		err = i.openLinux()
	case "darwin":
		err = i.openDarwin()
	case "windows":
		err = i.openWindows()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err != nil {
		return fmt.Errorf("failed to open interface: %w", err)
	}

	i.isOpen = true
	go i.packetProcessor()
	return nil
}

// Close closes the TUN interface
func (i *Interface) Close() error {
	if !i.isOpen {
		return nil
	}

	i.isOpen = false
	close(i.readChan)
	close(i.writeChan)

	if i.file != nil {
		return i.file.Close()
	}
	return nil
}

// Read reads a packet from the TUN interface
func (i *Interface) Read() ([]byte, error) {
	if !i.isOpen {
		return nil, fmt.Errorf("interface not open")
	}

	select {
	case packet := <-i.readChan:
		return packet, nil
	default:
		return nil, fmt.Errorf("no packet available")
	}
}

// Write writes a packet to the TUN interface
func (i *Interface) Write(packet []byte) error {
	if !i.isOpen {
		return fmt.Errorf("interface not open")
	}

	select {
	case i.writeChan <- packet:
		return nil
	default:
		return fmt.Errorf("write buffer is full")
	}
}

// GetIP returns the interface IP address
func (i *Interface) GetIP() net.IP {
	return i.IP
}

// GetNetmask returns the interface netmask
func (i *Interface) GetNetmask() net.IPMask {
	return i.Netmask
}

// GetGateway returns the interface gateway
func (i *Interface) GetGateway() net.IP {
	return i.Gateway
}

// GetMTU returns the interface MTU
func (i *Interface) GetMTU() int {
	return i.MTU
}

// IsOpen returns whether the interface is open
func (i *Interface) IsOpen() bool {
	return i.isOpen
}

// packetProcessor handles packet reading and writing in a separate goroutine
func (i *Interface) packetProcessor() {
	//@TODO: Implement packet processing logic
}

func (i *Interface) openLinux() error {
	//@TODO: Implement Linux TUN interface creation
	return fmt.Errorf("Linux TUN interface not yet implemented")
}

func (i *Interface) openDarwin() error {
	//@TODO: Implement macOS utun interface creation
	return fmt.Errorf("macOS utun interface not yet implemented")
}

func (i *Interface) openWindows() error {
	//@TODO: Implement Windows TAP interface creation
	return fmt.Errorf("Windows TAP interface not yet implemented")
}
